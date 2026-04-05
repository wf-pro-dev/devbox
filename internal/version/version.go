package version

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/wf-pro-dev/devbox/internal/db"
	"github.com/wf-pro-dev/devbox/internal/storage"
)

// Service handles versioning operations.
type Service struct {
	queries     *db.Queries
	blobs       *storage.BlobStore
	maxVersions int
}

const DefaultMaxVersions = 10

// Result describes the outcome of an Update call.
type Result int

const (
	ResultUpdated   Result = iota // content changed, new version created
	ResultUnchanged               // sha256 matched, nothing written
)

func (r Result) String() string {
	if r == ResultUpdated {
		return "updated"
	}
	return "unchanged"
}

// New creates a Service. maxVersions <= 0 uses DefaultMaxVersions.
func New(queries *db.Queries, blobs *storage.BlobStore, maxVersions int) *Service {
	if maxVersions <= 0 {
		maxVersions = DefaultMaxVersions
	}
	return &Service{queries: queries, blobs: blobs, maxVersions: maxVersions}
}

// UpdateParams is the input to Update.
type UpdateParams struct {
	FileID     string
	NewContent io.Reader
	UploadedBy string
	Message    string // optional commit-style message
}

// Update applies a content change to a file if the content has changed.
//
// Flow:
//  1. Write incoming bytes to the blob store (deduplicates by sha256).
//  2. If sha256 matches current file — return ResultUnchanged, no DB writes.
//  3. Snapshot current state as a versions row (no blob copy — CAS).
//  4. Update files row to point at new sha256, bump version number.
//  5. Prune versions beyond maxVersions.
func (s *Service) Update(ctx context.Context, p UpdateParams) (Result, db.File, error) {
	file, err := s.getFile(ctx, p.FileID)
	if err != nil {
		return 0, db.File{}, fmt.Errorf("get file: %w", err)
	}

	wr, err := s.blobs.Write(ctx, p.NewContent)
	if err != nil {
		return 0, db.File{}, fmt.Errorf("write blob: %w", err)
	}

	if wr.SHA256 == file.Sha256 {
		return ResultUnchanged, file, nil
	}

	if _, err := s.queries.SnapshotVersion(ctx, db.SnapshotVersionParams{
		FileID:     p.FileID,
		Version:    file.Version + 1,
		Sha256:     wr.SHA256,
		Size:       wr.Size,
		UploadedBy: p.UploadedBy,
		Message:    p.Message,
	}); err != nil {
		return 0, db.File{}, fmt.Errorf("snapshot version: %w", err)
	}

	updated, err := s.queries.UpdateFileContent(ctx, db.UpdateFileContentParams{
		ID:         p.FileID,
		Sha256:     wr.SHA256,
		Size:       wr.Size,
		Version:    file.Version + 1,
		UploadedBy: p.UploadedBy,
	})
	if err != nil {
		return 0, db.File{}, fmt.Errorf("update file record: %w", err)
	}

	go s.pruneVersions(context.Background(), p.FileID)

	return ResultUpdated, updated, nil
}

// Rollback restores a file to a previous version.
// Implemented as a forward update so no history is lost — current state is
// snapshotted before the target version's blob is restored.
func (s *Service) Rollback(ctx context.Context, fileID string, targetVersion int64, uploadedBy string) (db.File, error) {

	target, err := s.queries.GetVersion(ctx, db.GetVersionParams{
		FileID:  fileID,
		Version: targetVersion,
	})
	if err != nil {
		return db.File{}, fmt.Errorf("get version: %w", err)
	}

	rollbackVersions, err := s.queries.ListRollbackVersions(ctx, db.ListRollbackVersionsParams{
		FileID:  fileID,
		Version: target.Version,
	})
	if err != nil {
		return db.File{}, fmt.Errorf("list rollback versions: %w", err)
	}

	if err := s.queries.RollbackToVersion(ctx, db.RollbackToVersionParams{
		FileID:  fileID,
		Version: target.Version,
	}); err != nil {
		return db.File{}, fmt.Errorf("rollback to version: %w", err)
	}

	updated, err := s.queries.UpdateFileContent(ctx, db.UpdateFileContentParams{
		ID:         fileID,
		Sha256:     target.Sha256,
		Size:       target.Size,
		Version:    target.Version,
		UploadedBy: uploadedBy,
	})
	if err != nil {
		return db.File{}, fmt.Errorf("update file for rollback: %w", err)
	}

	go func() {
		for _, v := range rollbackVersions {
			if err := s.blobs.DeleteIfUnreferenced(ctx, v.Sha256); err != nil {
				log.Printf("version rollback: cleanup blob %s: %v", v.Sha256[:8], err)
			}
		}
	}()

	return updated, nil
}

// getFile fetches a single file by ID using GetFiles (the canonical point query).
func (s *Service) getFile(ctx context.Context, fileID string) (db.File, error) {
	files, err := s.queries.GetFiles(ctx, []string{fileID})
	if err != nil {
		return db.File{}, err
	}
	if len(files) == 0 {
		return db.File{}, fmt.Errorf("file %s not found", fileID)
	}
	return files[0], nil
}

// pruneVersions deletes version rows beyond maxVersions and cleans orphaned blobs.
func (s *Service) pruneVersions(ctx context.Context, fileID string) {
	minKeepIntrface, err := s.queries.GetMinVersionToKeep(ctx, db.GetMinVersionToKeepParams{
		FileID: fileID,
		Limit:  int64(s.maxVersions),
	})
	minKeep, ok := minKeepIntrface.(int64)
	if !ok {
		return
	}
	if err != nil || minKeep == 0 {
		return
	}

	prunable, err := s.queries.ListPrunableVersions(ctx, db.ListPrunableVersionsParams{
		FileID:  fileID,
		Version: minKeep,
	})
	if err != nil {
		log.Printf("version prune: list prunable for %s: %v", fileID, err)
		return
	}

	if err := s.queries.PruneOldVersions(ctx, db.PruneOldVersionsParams{
		FileID:  fileID,
		Version: minKeep,
	}); err != nil {
		log.Printf("version prune: delete rows for %s: %v", fileID, err)
		return
	}

	for _, v := range prunable {
		if err := s.blobs.DeleteIfUnreferenced(ctx, v.Sha256); err != nil {
			log.Printf("version prune: cleanup blob %s: %v", v.Sha256[:8], err)
		}
	}
}

// ── Internal helpers ───────────────────────────────────────────────────────────

func HashFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// StripV removes a leading "v" from a version string like "v3" -> "3".
func StripV(s string) string {
	if len(s) > 0 && (s[0] == 'v' || s[0] == 'V') {
		return s[1:]
	}
	return s
}

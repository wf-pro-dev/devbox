package version

import (
	"context"
	"fmt"
	"io"
	"log"

	"github.com/wf-pro-dev/devbox/internal/db"
	"github.com/wf-pro-dev/devbox/internal/storage"
)

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

// Service handles versioning operations.
type Service struct {
	queries     *db.Queries
	blobs       *storage.BlobStore
	maxVersions int
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
		Version:    file.Version,
		Sha256:     file.Sha256,
		Size:       file.Size,
		UploadedBy: file.UploadedBy,
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
	// Find the target version in the version list.
	versions, err := s.queries.ListVersions(ctx, db.ListVersionsParams{
		FileID: &fileID,
	})
	if err != nil {
		return db.File{}, fmt.Errorf("list versions: %w", err)
	}
	var target *db.Version
	for i := range versions {
		if versions[i].Version == targetVersion {
			target = &versions[i]
			break
		}
	}
	if target == nil {
		return db.File{}, fmt.Errorf("version %d not found", targetVersion)
	}

	file, err := s.getFile(ctx, fileID)
	if err != nil {
		return db.File{}, fmt.Errorf("get file: %w", err)
	}

	latestNumIntrface, err := s.queries.GetLatestVersionNumber(ctx, fileID)
	if err != nil {
		return db.File{}, fmt.Errorf("get latest version: %w", err)
	}
	latestNum, ok := latestNumIntrface.(int64)
	if !ok {
		return db.File{}, fmt.Errorf("get latest version: %w", err)
	}

	if _, err := s.queries.SnapshotVersion(ctx, db.SnapshotVersionParams{
		FileID:     fileID,
		Version:    latestNum,
		Sha256:     file.Sha256,
		Size:       file.Size,
		UploadedBy: file.UploadedBy,
		Message:    fmt.Sprintf("pre-rollback snapshot (was v%d)", latestNum),
	}); err != nil {
		return db.File{}, fmt.Errorf("snapshot before rollback: %w", err)
	}

	updated, err := s.queries.UpdateFileContent(ctx, db.UpdateFileContentParams{
		ID:         fileID,
		Sha256:     target.Sha256,
		Size:       target.Size,
		Version:    latestNum + 1,
		UploadedBy: uploadedBy,
	})
	if err != nil {
		return db.File{}, fmt.Errorf("update file for rollback: %w", err)
	}

	go s.pruneVersions(context.Background(), fileID)

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

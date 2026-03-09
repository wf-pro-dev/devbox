// Package version implements file versioning logic for devbox.
// It handles sha256-based change detection, snapshotting old versions,
// updating file content, and pruning old versions.
package version

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/wf-pro-dev/devbox/internal/db"
	"github.com/wf-pro-dev/devbox/internal/storage"
)

const DefaultMaxVersions = 10

// UpdateResult describes the outcome of a single file update attempt.
type UpdateResult int

const (
	ResultUpdated   UpdateResult = iota // new version created
	ResultUnchanged                     // sha256 matched, nothing done
)

func (r UpdateResult) String() string {
	switch r {
	case ResultUpdated:
		return "updated"
	case ResultUnchanged:
		return "unchanged"
	default:
		return "unknown"
	}
}

// Service handles versioning operations.
type Service struct {
	queries     *db.Queries
	blobs       *storage.BlobStore
	maxVersions int
}

// New creates a new versioning Service.
func New(queries *db.Queries, blobs *storage.BlobStore, maxVersions int) *Service {
	if maxVersions <= 0 {
		maxVersions = DefaultMaxVersions
	}
	return &Service{queries: queries, blobs: blobs, maxVersions: maxVersions}
}

// UpdateParams holds the inputs for a file content update.
type UpdateParams struct {
	FileID     string
	NewContent io.Reader
	UploadedBy string
	Message    string
}

// Update applies a content update to a file if the content has changed.
// Returns ResultUnchanged if sha256 matches — no DB writes occur.
// Returns ResultUpdated if content changed — old version is snapshotted,
// file record updated, and old versions pruned to maxVersions.
func (s *Service) Update(ctx context.Context, p UpdateParams) (UpdateResult, db.File, error) {
	// Fetch current file state.
	file, err := s.queries.GetFile(ctx, p.FileID)
	if err != nil {
		return 0, db.File{}, fmt.Errorf("get file: %w", err)
	}

	// Write incoming content to a temp blob to compute sha256.
	tmpID := p.FileID + "-tmp"
	size, newSHA256, err := s.blobs.Write(tmpID, p.NewContent)
	if err != nil {
		return 0, db.File{}, fmt.Errorf("write temp blob: %w", err)
	}

	// No change — clean up temp blob and return early.
	if newSHA256 == file.Sha256 {
		s.blobs.Delete(tmpID)
		return ResultUnchanged, file, nil
	}

	// Content changed — snapshot the current version first.
	latestNumIntf, err := s.queries.GetLatestVersionNumber(ctx, p.FileID)
	if err != nil {
		s.blobs.Delete(tmpID)
		return 0, db.File{}, fmt.Errorf("get latest version number: %w", err)
	}
	latestNum := latestNumIntf.(int64)
	nextNum := latestNum + 1

	// Copy current blob to a stable versioned path before overwriting.
	versionBlobID := fmt.Sprintf("%s-v%d", p.FileID, latestNum)
	if err := copyBlob(s.blobs, p.FileID, versionBlobID); err != nil {
		s.blobs.Delete(tmpID)
		return 0, db.File{}, fmt.Errorf("snapshot blob: %w", err)
	}

	// Record the snapshot in the versions table.
	if _, err := s.queries.SnapshotVersion(ctx, db.SnapshotVersionParams{
		FileID:        p.FileID,
		VersionNumber: nextNum,
		BlobPath:      s.blobs.BlobPath(versionBlobID),
		Sha256:        file.Sha256,
		Size:          file.Size,
		UploadedBy:    file.UploadedBy,
		Message:       p.Message,
	}); err != nil {
		s.blobs.Delete(tmpID)
		s.blobs.Delete(versionBlobID)
		return 0, db.File{}, fmt.Errorf("snapshot version: %w", err)
	}

	// Promote temp blob to the canonical file blob path.
	if err := s.blobs.Replace(p.FileID, tmpID); err != nil {
		s.blobs.Delete(tmpID)
		return 0, db.File{}, fmt.Errorf("promote blob: %w", err)
	}

	// Update the file record with new content metadata.
	updated, err := s.queries.UpdateFileContent(ctx, db.UpdateFileContentParams{
		ID:         p.FileID,
		BlobPath:   s.blobs.BlobPath(p.FileID),
		Sha256:     newSHA256,
		Size:       size,
		Version:    nextNum,
		UploadedBy: p.UploadedBy,
	})
	if err != nil {
		return 0, db.File{}, fmt.Errorf("update file record: %w", err)
	}

	// Prune versions beyond maxVersions.
	s.pruneVersions(ctx, p.FileID)

	return ResultUpdated, updated, nil
}

// pruneVersions deletes version rows and blobs beyond s.maxVersions.
// Strategy: find the minimum version_number among the N most recent,
// then delete everything older than that cutoff.
func (s *Service) pruneVersions(ctx context.Context, fileID string) {
	// Find the cutoff: lowest version_number we want to keep.
	minKeepIntf, err := s.queries.GetMinVersionToKeep(ctx, db.GetMinVersionToKeepParams{
		FileID: fileID,
		Limit:  int64(s.maxVersions),
	})
	if err != nil {
		log.Printf("version prune: get min version for %s: %v", fileID, err)
		return
	}
	minKeep := minKeepIntf.(int64)
	// Nothing to prune if all versions fall within the keep window.
	if minKeep == 0 {
		return
	}

	// Collect blob paths before deleting rows so we can clean up disk.
	paths, err := s.queries.GetOldBlobPaths(ctx, db.GetOldBlobPathsParams{
		FileID:        fileID,
		VersionNumber: minKeep,
	})
	if err != nil {
		log.Printf("version prune: get old blob paths for %s: %v", fileID, err)
		return
	}

	for _, path := range paths {
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			log.Printf("version prune: delete blob %s: %v", path, err)
		}
	}

	if err := s.queries.PruneOldVersions(ctx, db.PruneOldVersionsParams{
		FileID:        fileID,
		VersionNumber: minKeep,
	}); err != nil {
		log.Printf("version prune: delete rows for %s: %v", fileID, err)
	}
}

// copyBlob copies the blob at srcID to dstID in the blob store.
func copyBlob(blobs *storage.BlobStore, srcID, dstID string) error {
	r, err := blobs.Read(srcID)
	if err != nil {
		return fmt.Errorf("read src blob: %w", err)
	}
	defer r.Close()
	if _, _, err := blobs.Write(dstID, r); err != nil {
		return fmt.Errorf("write dst blob: %w", err)
	}
	return nil
}

package models

import (
	"context"
	"io"
	"log"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/wf-pro-dev/devbox/internal/auth"
	"github.com/wf-pro-dev/devbox/internal/db"
	"github.com/wf-pro-dev/devbox/internal/search"
	"github.com/wf-pro-dev/devbox/internal/storage"
	"github.com/wf-pro-dev/devbox/types"
)

func CreateFile(ctx context.Context, store *storage.Store, blobs *storage.BlobStore, searcher *search.Searcher, content io.Reader, filePath string, description string, language string, tags []string) (*types.File, error) {

	wr, err := blobs.Write(ctx, content)
	if err != nil {
		return nil, err
	}

	fileID := uuid.New().String()
	file, err := store.Queries.CreateFile(ctx, db.CreateFileParams{
		ID:          fileID,
		Path:        filePath,
		FileName:    filepath.Base(filePath),
		Description: description,
		Language:    language,
		Size:        wr.Size,
		Sha256:      wr.SHA256,
		UploadedBy:  callerHost(ctx),
	})
	if err != nil {
		return nil, err
	}

	// Snapshot First Version
	if _, err := store.Queries.SnapshotVersion(ctx, db.SnapshotVersionParams{
		FileID:     fileID,
		Version:    1,
		Sha256:     wr.SHA256,
		Size:       wr.Size,
		UploadedBy: callerHost(ctx),
		Message:    "Initial Version",
	}); err != nil {
		return nil, err
	}

	if err := ApplyTags(ctx, store.Queries, fileID, tags); err != nil {
		return nil, err
	}

	m := BuildTagMap(ctx, store.Queries, []string{file.ID})

	go IndexContent(ctx, blobs, searcher, file.ID, wr.SHA256)

	return &types.File{File: file, Tags: m[file.ID]}, nil
}

// Helper functions

func IndexContent(ctx context.Context, blobs *storage.BlobStore, searcher *search.Searcher, fileID, sha256hex string) {
	rc, err := blobs.Open(sha256hex)
	if err != nil {
		log.Printf("index: open blob %s: %v", fileID, err)
		return
	}
	defer rc.Close()
	content, err := io.ReadAll(rc)
	if err != nil {
		log.Printf("index: read blob %s: %v", fileID, err)
		return
	}
	_ = searcher.IndexFileContent(ctx, fileID, string(content))
}

// applyTags upserts each tag and attaches it to fileID.
func ApplyTags(ctx context.Context, q *db.Queries, fileID string, tagNames []string) error {
	for _, name := range tagNames {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		tag, err := q.UpsertTag(ctx, name)
		if err != nil {
			return err
		}
		if err := q.AddTagToFile(ctx, db.AddTagToFileParams{
			FileID: fileID,
			TagID:  tag.ID,
		}); err != nil {
			return err
		}
	}
	return nil
}

func SplitTags(raw string) []string {
	if raw == "" {
		return nil
	}
	var out []string
	for _, t := range strings.Split(raw, ",") {
		if t = strings.TrimSpace(t); t != "" {
			out = append(out, t)
		}
	}
	return out
}

// buildTagMap fetches tags for a set of file IDs in one query.
// Returns map[fileID][]tagName.
func BuildTagMap(ctx context.Context, q *db.Queries, ids []string) map[string][]string {
	if len(ids) == 0 {
		return map[string][]string{}
	}
	rows, _ := q.ListTagsForFiles(ctx, ids)
	out := make(map[string][]string, len(ids))
	for _, row := range rows {
		out[row.FileID] = append(out[row.FileID], row.Name)
	}
	return out
}

func callerHost(ctx context.Context) string {
	if id, ok := auth.FromContext(ctx); ok {
		return id.Hostname
	}
	return "unknown"
}

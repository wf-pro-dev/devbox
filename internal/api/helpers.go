package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/wf-pro-dev/devbox/internal/db"
	"github.com/wf-pro-dev/devbox/types"
)

// ── JSON helpers ──────────────────────────────────────────────────────────────

func jsonOK(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

func jsonCreated(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(v)
}

func jsonError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

func jsonNoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

// ── Response types ────────────────────────────────────────────────────────────

// dirResponse describes a virtual directory derived from a path prefix.
type dirResponse struct {
	Prefix    string       `json:"prefix"`
	FileCount int          `json:"file_count"`
	Tags      []string     `json:"tags"`
	Files     []types.File `json:"files,omitempty"`
}

// ── Tag helpers ───────────────────────────────────────────────────────────────

// applyTags upserts each tag and attaches it to fileID.
func applyTags(ctx context.Context, q *db.Queries, fileID string, tagNames []string) error {
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

// buildTagMap fetches tags for a set of file IDs in one query.
// Returns map[fileID][]tagName.
func buildTagMap(ctx context.Context, q *db.Queries, ids []string) map[string][]string {
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

// prefixTags returns distinct tag names across all files under prefix.
func prefixTags(ctx context.Context, q *db.Queries, prefix string) []string {
	files, _ := q.ListFiles(ctx, db.ListFilesParams{Prefix: &prefix})
	if len(files) == 0 {
		return []string{}
	}
	ids := make([]string, len(files))
	for i, f := range files {
		ids[i] = f.ID
	}
	rows, _ := q.ListTagsForFiles(ctx, ids)
	seen := make(map[string]struct{})
	var names []string
	for _, row := range rows {
		if _, ok := seen[row.Name]; !ok {
			seen[row.Name] = struct{}{}
			names = append(names, row.Name)
		}
	}
	return names
}

// ── Misc ──────────────────────────────────────────────────────────────────────

func orEmpty(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

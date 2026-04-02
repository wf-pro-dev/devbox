package api

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/wf-pro-dev/devbox/internal/db"
	"github.com/wf-pro-dev/devbox/internal/models"
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

func buildTagMap(ctx context.Context, q *db.Queries, ids []string) map[string][]string {
	return models.BuildTagMap(ctx, q, ids)
}

func applyTags(ctx context.Context, q *db.Queries, fileID string, tagNames []string) error {
	return models.ApplyTags(ctx, q, fileID, tagNames)
}

func splitTags(raw string) []string {
	return models.SplitTags(raw)
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

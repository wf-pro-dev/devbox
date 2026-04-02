package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/wf-pro-dev/devbox/internal/auth"
	"github.com/wf-pro-dev/devbox/internal/db"
	"github.com/wf-pro-dev/devbox/internal/models"
	"github.com/wf-pro-dev/devbox/internal/search"
	"github.com/wf-pro-dev/devbox/internal/storage"
	"github.com/wf-pro-dev/devbox/internal/version"
	"github.com/wf-pro-dev/devbox/types"
)

const maxUploadSize = 100 << 20 // 100 MB

type filesHandler struct {
	store    *storage.Store
	blobs    *storage.BlobStore
	searcher *search.Searcher
	verSvc   *version.Service
}

// ── GET /files ────────────────────────────────────────────────────────────────
// Query params (all optional, combinable):
//   ?q=<fts>      full-text search (handled separately, bypasses ListFiles)
//   ?dir=<prefix> filter by path prefix
//   ?tag=<name>   filter by tag
//   ?lang=<lang>  filter by language

func (h *filesHandler) handleList(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	q := r.URL.Query()

	// FTS is a separate code path — cannot be expressed as a SQL filter.
	if fts := q.Get("q"); fts != "" {
		files, err := h.searcher.Search(ctx, fts)
		if err != nil {
			jsonError(w, "search failed", http.StatusInternalServerError)
			log.Printf("search files: %v", err)
			return
		}
		jsonOK(w, files)
		return
	}

	params := db.ListFilesParams{}
	if v := q.Get("dir"); v != "" {
		p := toPrefix(v)
		params.Prefix = &p
	}
	if v := q.Get("tag"); v != "" {
		params.Tag = &v
	}
	if v := q.Get("lang"); v != "" {
		params.Lang = &v
	}

	fs, err := h.store.Queries.ListFiles(ctx, params)
	if err != nil {
		jsonError(w, "failed to list files", http.StatusInternalServerError)
		log.Printf("list files: %v", err)
		return
	}

	ids := make([]string, len(fs))
	for i, f := range fs {
		ids[i] = f.ID
	}

	var files []types.File = []types.File{}
	tagMap := buildTagMap(ctx, h.store.Queries, ids)
	for _, f := range fs {
		files = append(files, types.File{File: f, Tags: tagMap[f.ID]})
	}

	jsonOK(w, files)
}

// ── GET /files/{id} ───────────────────────────────────────────────────────────
// ?meta=true returns JSON metadata instead of the raw blob.

func (h *filesHandler) handleGet(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	file, err := h.store.ResolveFile(r.PathValue("id"))
	if err != nil {
		jsonError(w, err.Error(), http.StatusNotFound)
		return
	}

	m := buildTagMap(ctx, h.store.Queries, []string{file.ID})
	if r.URL.Query().Get("meta") == "true" {
		jsonOK(w, types.File{File: *file, Tags: m[file.ID]})
		return
	}

	blob, err := h.blobs.Open(file.Sha256)
	if err != nil {
		jsonError(w, "blob not found", http.StatusNotFound)
		log.Printf("open blob %s: %v", file.ID, err)
		return
	}
	defer blob.Close()

	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, file.FileName))
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("X-File-Name", file.FileName)
	w.Header().Set("X-File-Path", file.Path)
	w.Header().Set("X-File-SHA256", file.Sha256)
	w.Header().Set("X-File-Version", strconv.FormatInt(file.Version, 10))
	w.Header().Set("Content-Length", strconv.FormatInt(file.Size, 10))

	if _, err := io.Copy(w, blob); err != nil {
		log.Printf("stream blob %s: %v", file.ID, err)
	}
}

// ── POST /files ───────────────────────────────────────────────────────────────
// Multipart fields: file (required), path, description, language, tags.

func (h *filesHandler) handleUpload(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)

	if err := r.ParseMultipartForm(10 << 20); err != nil {
		jsonError(w, "request too large or invalid multipart form", http.StatusBadRequest)
		return
	}

	formFile, header, err := r.FormFile("file")
	if err != nil {
		jsonError(w, "missing 'file' field", http.StatusBadRequest)
		return
	}
	defer formFile.Close()

	filePath := r.FormValue("path")
	if filePath == "" {
		filePath = header.Filename
	}

	language := r.FormValue("language")
	if language == "" {
		language = detectLanguage(header.Filename)
	}

	tags := splitTags(r.FormValue("tags"))

	file, err := models.CreateFile(ctx, h.store, h.blobs, h.searcher,
		formFile,
		filePath,
		r.FormValue("description"),
		language,
		tags,
	)
	if err != nil {
		jsonError(w, "failed to create file", http.StatusInternalServerError)
		return
	}

	jsonCreated(w, file)
}

// ── PUT /files/{id} ───────────────────────────────────────────────────────────
// Updates file content. Creates a new version when content has changed.
// Multipart fields: file (required), message (optional).

func (h *filesHandler) handleUpdate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	file, err := h.store.ResolveFile(r.PathValue("id"))
	if err != nil {
		jsonError(w, err.Error(), http.StatusNotFound)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		jsonError(w, "invalid multipart form", http.StatusBadRequest)
		return
	}

	formFile, _, err := r.FormFile("file")
	if err != nil {
		jsonError(w, "missing 'file' field", http.StatusBadRequest)
		return
	}
	defer formFile.Close()

	result, updated, err := h.verSvc.Update(ctx, version.UpdateParams{
		FileID:     file.ID,
		NewContent: formFile,
		UploadedBy: callerHost(ctx),
		Message:    r.FormValue("message"),
	})
	if err != nil {
		jsonError(w, "update failed", http.StatusInternalServerError)
		log.Printf("update file %s: %v", file.ID, err)
		return
	}

	if result == version.ResultUpdated {
		go models.IndexContent(ctx, h.blobs, h.searcher, file.ID, updated.Sha256)
	}

	m := buildTagMap(ctx, h.store.Queries, []string{updated.ID})

	jsonOK(w, map[string]any{
		"result": result.String(),
		"file":   types.File{File: updated, Tags: m[updated.ID]},
	})
}

// ── PATCH /files/{id} ─────────────────────────────────────────────────────────
// Edits metadata: description, language, path (rename/move).
// JSON body fields are all optional — only provided fields are updated.

func (h *filesHandler) handleEditMeta(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	file, err := h.store.ResolveFile(r.PathValue("id"))
	if err != nil {
		jsonError(w, err.Error(), http.StatusNotFound)
		return
	}

	var body struct {
		Description *string `json:"description"`
		Language    *string `json:"language"`
		Path        *string `json:"path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonError(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	// Apply path rename first so UpdateFileMeta sees the right row.
	if body.Path != nil && *body.Path != file.Path {
		if _, err := h.store.Queries.MoveFile(ctx, db.MoveFileParams{
			ID:       file.ID,
			Path:     *body.Path,
			FileName: filepath.Base(*body.Path),
		}); err != nil {
			jsonError(w, "rename failed", http.StatusInternalServerError)
			log.Printf("move file %s: %v", file.ID, err)
			return
		}
	}

	desc := file.Description
	if body.Description != nil {
		desc = *body.Description
	}
	lang := file.Language
	if body.Language != nil {
		lang = *body.Language
	}

	updated, err := h.store.Queries.UpdateFileMeta(ctx, db.UpdateFileMetaParams{
		ID:          file.ID,
		Description: desc,
		Language:    lang,
	})
	if err != nil {
		jsonError(w, "meta update failed", http.StatusInternalServerError)
		return
	}
	m := buildTagMap(ctx, h.store.Queries, []string{updated.ID})

	jsonOK(w, types.File{File: updated, Tags: m[updated.ID]})
}

// ── DELETE /files/{id} ────────────────────────────────────────────────────────

func (h *filesHandler) handleDelete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	file, err := h.store.ResolveFile(r.PathValue("id"))
	if err != nil {
		jsonError(w, err.Error(), http.StatusNotFound)
		return
	}

	sha := file.Sha256
	if err := h.store.Queries.DeleteFile(ctx, file.ID); err != nil {
		jsonError(w, "delete failed", http.StatusInternalServerError)
		log.Printf("delete file %s: %v", file.ID, err)
		return
	}

	go h.blobs.DeleteIfUnreferenced(ctx, sha)
	jsonNoContent(w)
}

// ── POST /files/{id}/tags ─────────────────────────────────────────────────────
// JSON body: {"tags": ["tag1", "tag2"]}

func (h *filesHandler) handleAddTags(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	file, err := h.store.ResolveFile(r.PathValue("id"))
	if err != nil {
		jsonError(w, err.Error(), http.StatusNotFound)
		return
	}

	var body struct {
		Tags []string `json:"tags"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonError(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	if err := applyTags(ctx, h.store.Queries, file.ID, body.Tags); err != nil {
		jsonError(w, "failed to apply tags", http.StatusInternalServerError)
		return
	}

	m := buildTagMap(ctx, h.store.Queries, []string{file.ID})
	jsonOK(w, map[string][]string{"tags": m[file.ID]})
}

// ── DELETE /files/{id}/tags/{tag} ─────────────────────────────────────────────

func (h *filesHandler) handleRemoveTag(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	file, err := h.store.ResolveFile(r.PathValue("id"))
	if err != nil {
		jsonError(w, err.Error(), http.StatusNotFound)
		return
	}

	tag, err := h.store.Queries.GetTagByName(ctx, r.PathValue("tag"))
	if isNotFound(err) {
		jsonError(w, "tag not found", http.StatusNotFound)
		return
	}
	if err != nil {
		jsonError(w, "failed to get tag", http.StatusInternalServerError)
		return
	}

	h.store.Queries.RemoveTagFromFile(ctx, db.RemoveTagFromFileParams{
		FileID: file.ID,
		TagID:  tag.ID,
	})
	jsonNoContent(w)
}

// ── POST /files/{id}/copy ─────────────────────────────────────────────────────
// Creates a new file record pointing at the same blob — no disk copy.
// JSON body: {"path": "new/path.sh"}

func (h *filesHandler) handleCopy(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	src, err := h.store.ResolveFile(r.PathValue("id"))
	if err != nil {
		jsonError(w, err.Error(), http.StatusNotFound)
		return
	}

	var body struct {
		Path string `json:"path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Path == "" {
		jsonError(w, "JSON body with 'path' is required", http.StatusBadRequest)
		return
	}

	newFile, err := h.store.Queries.CreateFile(ctx, db.CreateFileParams{
		ID:          uuid.New().String(),
		Path:        body.Path,
		FileName:    filepath.Base(body.Path),
		Description: src.Description,
		Language:    src.Language,
		Size:        src.Size,
		Sha256:      src.Sha256, // same blob — ref count bumped by trigger
		UploadedBy:  callerHost(ctx),
	})
	if err != nil {
		jsonError(w, "copy failed", http.StatusInternalServerError)
		log.Printf("copy file %s -> %s: %v", src.ID, body.Path, err)
		return
	}

	m := buildTagMap(ctx, h.store.Queries, []string{newFile.ID})
	jsonCreated(w, types.File{File: newFile, Tags: m[newFile.ID]})
}

// ── POST /files/{id}/move ─────────────────────────────────────────────────────
// JSON body: {"path": "new/path.sh"}

func (h *filesHandler) handleMove(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	file, err := h.store.ResolveFile(r.PathValue("id"))
	if err != nil {
		jsonError(w, err.Error(), http.StatusNotFound)
		return
	}

	var body struct {
		Path string `json:"path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Path == "" {
		jsonError(w, "JSON body with 'path' is required", http.StatusBadRequest)
		return
	}

	updated, err := h.store.Queries.MoveFile(ctx, db.MoveFileParams{
		ID:       file.ID,
		Path:     body.Path,
		FileName: filepath.Base(body.Path),
	})
	if err != nil {
		jsonError(w, "move failed", http.StatusInternalServerError)
		log.Printf("move file %s: %v", file.ID, err)
		return
	}

	m := buildTagMap(ctx, h.store.Queries, []string{updated.ID})
	jsonOK(w, types.File{File: updated, Tags: m[updated.ID]})
}

// ── GET /files/{id}/versions ──────────────────────────────────────────────────

func (h *filesHandler) handleListVersions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	file, err := h.store.ResolveFile(r.PathValue("id"))
	if err != nil {
		jsonError(w, err.Error(), http.StatusNotFound)
		return
	}

	versions, err := h.store.Queries.ListVersions(ctx, db.ListVersionsParams{
		FileID: &file.ID,
	})
	if err != nil {
		jsonError(w, "failed to list versions", http.StatusInternalServerError)
		return
	}

	jsonOK(w, versions)
}

// GET /files/{id}/versions/{n}
func (h *filesHandler) handleGetVersion(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	file, err := h.store.ResolveFile(r.PathValue("id"))
	if err != nil {
		jsonError(w, err.Error(), http.StatusNotFound)
		return
	}

	n, err := strconv.ParseInt(r.PathValue("n"), 10, 64)
	if err != nil {
		jsonError(w, "invalid version number", http.StatusBadRequest)
		return
	}

	versions, err := h.store.Queries.ListVersions(ctx, db.ListVersionsParams{
		FileID: &file.ID,
	})
	if err != nil {
		jsonError(w, "failed to list versions", http.StatusInternalServerError)
		return
	}

	if len(versions) == 0 {
		jsonError(w, "no versions found", http.StatusNotFound)
		return
	}

	var version db.Version
	for _, v := range versions {
		if v.Version == n {
			version = v
			break
		}
	}

	blob, err := h.blobs.Open(version.Sha256)
	if err != nil {
		jsonError(w, "blob not found", http.StatusNotFound)
		log.Printf("open blob %s: %v", file.ID, err)
		return
	}
	defer blob.Close()

	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, file.FileName))
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("X-File-Name", file.FileName)
	w.Header().Set("X-File-Path", file.Path)
	w.Header().Set("X-File-SHA256", version.Sha256)
	w.Header().Set("X-File-Version", strconv.FormatInt(version.Version, 10))

	if _, err := io.Copy(w, blob); err != nil {
		log.Printf("stream blob %s: %v", file.ID, err)
	}

}

// ── POST /files/{id}/versions/{n}/rollback ───────────────────────────────────

func (h *filesHandler) handleRollback(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	file, err := h.store.ResolveFile(r.PathValue("id"))
	if err != nil {
		jsonError(w, err.Error(), http.StatusNotFound)
		return
	}

	n, err := strconv.ParseInt(r.PathValue("n"), 10, 64)
	if err != nil {
		jsonError(w, "invalid version number", http.StatusBadRequest)
		return
	}

	updated, err := h.verSvc.Rollback(ctx, file.ID, n, callerHost(ctx))
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	m := buildTagMap(ctx, h.store.Queries, []string{updated.ID})
	jsonOK(w, types.File{File: updated, Tags: m[updated.ID]})
}

// ── GET /files/{id}/diff ──────────────────────────────────────────────────────
// Returns a metadata diff between two versions.
// Query params: ?a=<n>&b=<m> (defaults: a=current, b=previous)

func (h *filesHandler) handleDiff(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	file, err := h.store.ResolveFile(r.PathValue("id"))
	if err != nil {
		jsonError(w, err.Error(), http.StatusNotFound)
		return
	}

	versions, err := h.store.Queries.ListVersions(ctx, db.ListVersionsParams{
		FileID: &file.ID,
	})
	if err != nil || len(versions) == 0 {
		jsonOK(w, map[string]any{"message": "no version history"})
		return
	}

	// Build lookup: version_number → db.Version.
	// Also include current file state as a pseudo-version.
	byNum := make(map[int64]db.Version, len(versions)+1)
	for _, v := range versions {
		byNum[v.Version] = v
	}
	byNum[file.Version] = db.Version{
		FileID:     file.ID,
		Version:    file.Version,
		Sha256:     file.Sha256,
		Size:       file.Size,
		UploadedBy: file.UploadedBy,
		CreatedAt:  file.UpdatedAt,
		Message:    "(current)",
	}

	parseVer := func(s string, def int64) (db.Version, bool) {
		if s == "" {
			v, ok := byNum[def]
			return v, ok
		}
		n, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return db.Version{}, false
		}
		v, ok := byNum[n]
		return v, ok
	}

	vA, okA := parseVer(r.URL.Query().Get("a"), file.Version)
	vB, okB := parseVer(r.URL.Query().Get("b"), versions[0].Version)
	if !okA || !okB {
		jsonError(w, "version not found", http.StatusNotFound)
		return
	}

	jsonOK(w, map[string]any{
		"file": map[string]string{"id": file.ID, "path": file.Path},
		"a":    vA,
		"b":    vB,
		"changed": map[string]any{
			"sha256":     vA.Sha256 != vB.Sha256,
			"size_delta": vA.Size - vB.Size,
		},
	})
}

// ── Internal helpers ──────────────────────────────────────────────────────────

func callerHost(ctx context.Context) string {
	if id, ok := auth.FromContext(ctx); ok {
		return id.Hostname
	}
	return "unknown"
}

func detectLanguage(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".sh", ".bash":
		return "bash"
	case ".yaml", ".yml":
		return "yaml"
	case ".toml":
		return "toml"
	case ".json":
		return "json"
	case ".py":
		return "python"
	case ".go":
		return "go"
	case ".ts", ".tsx":
		return "typescript"
	case ".js", ".jsx":
		return "javascript"
	case ".sql":
		return "sql"
	case ".service", ".timer", ".socket":
		return "systemd"
	case ".conf", ".ini", ".cfg":
		return "ini"
	case ".md":
		return "markdown"
	case ".rs":
		return "rust"
	case ".rb":
		return "ruby"
	default:
		if strings.HasPrefix(filepath.Base(filename), "Dockerfile") {
			return "dockerfile"
		}
		return "text"
	}
}

func isNotFound(err error) bool {
	return err != nil && (errors.Is(err, os.ErrNotExist) || strings.Contains(err.Error(), "no rows"))
}

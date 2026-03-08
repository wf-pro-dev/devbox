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
	"strings"

	"github.com/google/uuid"
	"github.com/wf-pro-dev/devbox/internal/auth"
	"github.com/wf-pro-dev/devbox/internal/db"
	"github.com/wf-pro-dev/devbox/internal/search"
	"github.com/wf-pro-dev/devbox/internal/storage"
)

const maxUploadSize = 100 << 20 // 100 MB

// fileResponse is the JSON shape returned for a single file.
type fileResponse struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Language    string   `json:"language"`
	Size        int64    `json:"size"`
	SHA256      string   `json:"sha256"`
	UploadedBy  string   `json:"uploaded_by"`
	CreatedAt   string   `json:"created_at"`
	UpdatedAt   string   `json:"updated_at"`
	Tags        []string `json:"tags"`
}

func fileToResponse(f db.File, tags []string) fileResponse {
	if tags == nil {
		tags = []string{}
	}
	return fileResponse{
		ID:          f.ID,
		Name:        f.Name,
		Description: f.Description,
		Language:    f.Language,
		Size:        f.Size,
		SHA256:      f.Sha256,
		UploadedBy:  f.UploadedBy,
		CreatedAt:   f.CreatedAt,
		UpdatedAt:   f.UpdatedAt,
		Tags:        tags,
	}
}

// filesHandler groups the dependencies for file-related handlers.
type filesHandler struct {
	queries  *db.Queries
	blobs    *storage.BlobStore
	searcher *search.Searcher
}

// tagsForFiles fetches tags for a slice of files and returns a map of
// fileID → tag names.
func (h *filesHandler) tagsForFiles(ctx context.Context, files []db.File) map[string][]string {
	result := make(map[string][]string, len(files))
	for _, f := range files {
		tags, err := h.queries.ListTagsForFile(ctx, f.ID)
		if err != nil {
			result[f.ID] = []string{}
			continue
		}
		names := make([]string, len(tags))
		for i, t := range tags {
			names[i] = t.Name
		}
		result[f.ID] = names
	}
	return result
}

// handleListFiles handles GET /files
// Supports ?tag=bash and ?q=search+term query params.
func (h *filesHandler) handleListFiles(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	tag := r.URL.Query().Get("tag")
	q := r.URL.Query().Get("q")

	var (
		files []db.File
		err   error
	)

	switch {
	case q != "":
		files, err = h.searcher.SearchFiles(ctx, q)
	case tag != "":
		files, err = h.queries.ListFilesForTag(ctx, tag)
	default:
		files, err = h.queries.ListFiles(ctx)
	}

	if err != nil {
		jsonError(w, "failed to list files", http.StatusInternalServerError)
		log.Printf("ListFiles: %v", err)
		return
	}

	tagsMap := h.tagsForFiles(ctx, files)
	resp := make([]fileResponse, len(files))
	for i, f := range files {
		resp[i] = fileToResponse(f, tagsMap[f.ID])
	}

	jsonOK(w, resp)
}

// handleGetFile handles GET /files/{id}
func (h *filesHandler) handleGetFile(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := r.PathValue("id")

	file, err := h.queries.GetFile(ctx, id)
	if err != nil {
		if isNotFound(err) {
			jsonError(w, "file not found", http.StatusNotFound)
			return
		}
		jsonError(w, "failed to get file", http.StatusInternalServerError)
		log.Printf("GetFile %s: %v", id, err)
		return
	}

	// If ?meta=true return JSON metadata only.
	if r.URL.Query().Get("meta") == "true" {
		tags, _ := h.queries.ListTagsForFile(ctx, id)
		names := make([]string, len(tags))
		for i, t := range tags {
			names[i] = t.Name
		}
		jsonOK(w, fileToResponse(file, names))
		return
	}

	// Otherwise stream the raw blob.
	blob, err := h.blobs.Read(file.ID)
	if err != nil {
		jsonError(w, "blob not found", http.StatusNotFound)
		log.Printf("ReadBlob %s: %v", id, err)
		return
	}
	defer blob.Close()

	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, file.Name))
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("X-File-Name", file.Name)
	w.Header().Set("X-File-Language", file.Language)

	if _, err := io.Copy(w, blob); err != nil {
		log.Printf("stream blob %s: %v", id, err)
	}
}

// handleUploadFile handles POST /files
func (h *filesHandler) handleUploadFile(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)

	if err := r.ParseMultipartForm(10 << 20); err != nil {
		jsonError(w, "request too large or invalid multipart form", http.StatusBadRequest)
		return
	}

	formFile, header, err := r.FormFile("file")
	if err != nil {
		jsonError(w, "missing 'file' field in form", http.StatusBadRequest)
		return
	}
	defer formFile.Close()

	description := r.FormValue("description")
	language := r.FormValue("language")
	if language == "" {
		language = detectLanguage(header.Filename)
	}

	// Parse comma-separated tags e.g. tags=bash,deploy,cron
	var tagNames []string
	if raw := r.FormValue("tags"); raw != "" {
		for _, t := range strings.Split(raw, ",") {
			if t = strings.TrimSpace(t); t != "" {
				tagNames = append(tagNames, t)
			}
		}
	}

	uploadedBy := "unknown"
	if id, ok := auth.FromContext(ctx); ok {
		uploadedBy = id.Hostname
	}

	fileID := uuid.New().String()

	// Read content into memory for both blob write and FTS indexing.
	// For large files we write to disk and index separately.
	size, sha256hex, err := h.blobs.Write(fileID, formFile)
	if err != nil {
		jsonError(w, "failed to store file", http.StatusInternalServerError)
		log.Printf("WriteBlob %s: %v", fileID, err)
		return
	}

	file, err := h.queries.CreateFile(ctx, db.CreateFileParams{
		ID:          fileID,
		Name:        header.Filename,
		Description: description,
		Language:    language,
		Size:        size,
		BlobPath:    h.blobs.BlobPath(fileID),
		Sha256:      sha256hex,
		UploadedBy:  uploadedBy,
	})
	if err != nil {
		h.blobs.Delete(fileID)
		jsonError(w, "failed to save file metadata", http.StatusInternalServerError)
		log.Printf("CreateFile %s: %v", fileID, err)
		return
	}

	// Update FTS content with the actual file content.
	if err := h.searcher.UpdateContent(ctx, fileID, h.blobs); err != nil {
		// Non-fatal — file is stored, search just won't find content yet.
		log.Printf("FTS index %s: %v", fileID, err)
	}

	// Apply tags.
	appliedTags := h.applyTags(ctx, fileID, tagNames)

	w.WriteHeader(http.StatusCreated)
	jsonOK(w, fileToResponse(file, appliedTags))
}

// handleDeleteFile handles DELETE /files/{id}
func (h *filesHandler) handleDeleteFile(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	file, err := h.queries.GetFile(r.Context(), id)
	if err != nil {
		if isNotFound(err) {
			jsonError(w, "file not found", http.StatusNotFound)
			return
		}
		jsonError(w, "failed to get file", http.StatusInternalServerError)
		return
	}

	if err := h.queries.DeleteFile(r.Context(), id); err != nil {
		jsonError(w, "failed to delete file", http.StatusInternalServerError)
		log.Printf("DeleteFile %s: %v", id, err)
		return
	}

	if err := h.blobs.Delete(file.ID); err != nil {
		log.Printf("DeleteBlob %s: %v", id, err)
	}

	w.WriteHeader(http.StatusNoContent)
}

// handleAddTags handles POST /files/{id}/tags
// Body: {"tags": ["bash", "deploy"]}
func (h *filesHandler) handleAddTags(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := r.PathValue("id")

	if _, err := h.queries.GetFile(ctx, id); err != nil {
		if isNotFound(err) {
			jsonError(w, "file not found", http.StatusNotFound)
			return
		}
		jsonError(w, "failed to get file", http.StatusInternalServerError)
		return
	}

	var body struct {
		Tags []string `json:"tags"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonError(w, "invalid JSON body", http.StatusBadRequest)
		return
	}

	applied := h.applyTags(ctx, id, body.Tags)
	jsonOK(w, map[string][]string{"tags": applied})
}

// handleRemoveTag handles DELETE /files/{id}/tags/{tag}
func (h *filesHandler) handleRemoveTag(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := r.PathValue("id")
	tagName := r.PathValue("tag")

	tag, err := h.queries.GetTagByName(ctx, tagName)
	if err != nil {
		if isNotFound(err) {
			jsonError(w, "tag not found", http.StatusNotFound)
			return
		}
		jsonError(w, "failed to get tag", http.StatusInternalServerError)
		return
	}

	if err := h.queries.RemoveTagFromFile(ctx, db.RemoveTagFromFileParams{
		FileID: id,
		TagID:  tag.ID,
	}); err != nil {
		jsonError(w, "failed to remove tag", http.StatusInternalServerError)
		log.Printf("RemoveTag %s from %s: %v", tagName, id, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// applyTags upserts the given tag names and associates them with fileID.
// Returns the list of successfully applied tag names.
func (h *filesHandler) applyTags(ctx context.Context, fileID string, names []string) []string {
	var applied []string
	for _, name := range names {
		tag, err := h.queries.CreateTag(ctx, name)
		if err != nil {
			log.Printf("CreateTag %q: %v", name, err)
			continue
		}
		if err := h.queries.AddTagToFile(ctx, db.AddTagToFileParams{
			FileID: fileID,
			TagID:  tag.ID,
		}); err != nil {
			log.Printf("AddTagToFile %s → %s: %v", name, fileID, err)
			continue
		}
		applied = append(applied, name)
	}
	return applied
}

// detectLanguage guesses the language from the file extension.
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
	default:
		if strings.EqualFold(filename, "Dockerfile") {
			return "dockerfile"
		}
		return "text"
	}
}

func isNotFound(err error) bool {
	return err != nil && (errors.Is(err, os.ErrNotExist) || strings.Contains(err.Error(), "no rows"))
}

package api

import (
	"context"
	"log"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/wf-pro-dev/devbox/internal/auth"
	"github.com/wf-pro-dev/devbox/internal/db"
	"github.com/wf-pro-dev/devbox/internal/storage"
	ver "github.com/wf-pro-dev/devbox/internal/version"
)

type versionHandler struct {
	queries *db.Queries
	blobs   *storage.BlobStore
	version *ver.Service
}

// ── PUT /files/{id} ──────────────────────────────────────────────────────────

func (h *versionHandler) handleVersionFile(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := r.PathValue("id")

	r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		jsonError(w, "request too large or invalid multipart form", http.StatusBadRequest)
		return
	}

	formFile, _, err := r.FormFile("file")
	if err != nil {
		jsonError(w, "missing 'file' field in form", http.StatusBadRequest)
		return
	}
	defer formFile.Close()

	message := r.FormValue("message")
	uploadedBy := "unknown"
	if identity, ok := auth.FromContext(ctx); ok {
		uploadedBy = identity.Hostname
	}

	result, updated, err := h.version.Update(ctx, ver.UpdateParams{
		FileID:     id,
		NewContent: formFile,
		UploadedBy: uploadedBy,
		Message:    message,
	})
	if err != nil {
		if isNotFound(err) {
			jsonError(w, "file not found", http.StatusNotFound)
			return
		}
		jsonError(w, "failed to update file", http.StatusInternalServerError)
		log.Printf("UpdateFile %s: %v", id, err)
		return
	}

	tags, _ := h.queries.ListTagsForFile(ctx, id)
	names := make([]string, len(tags))
	for i, t := range tags {
		names[i] = t.Name
	}

	jsonOK(w, map[string]interface{}{
		"result": result.String(),
		"file":   fileToResponse(updated, names),
	})
}

// ── PUT /directories/{id} ────────────────────────────────────────────────────

type dirUpdateResult struct {
	Updated   []string `json:"updated"`
	Unchanged []string `json:"unchanged"`
	Added     []string `json:"added"`
}

func (h *versionHandler) handleUpdateDir(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := r.PathValue("id")

	dir, err := h.queries.GetDirectory(ctx, id)
	if err != nil {
		if isNotFound(err) {
			jsonError(w, "directory not found", http.StatusNotFound)
			return
		}
		jsonError(w, "failed to get directory", http.StatusInternalServerError)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 500<<20)
	if err := r.ParseMultipartForm(50 << 20); err != nil {
		jsonError(w, "request too large or invalid multipart form", http.StatusBadRequest)
		return
	}

	uploadedBy := "unknown"
	if identity, ok := auth.FromContext(ctx); ok {
		uploadedBy = identity.Hostname
	}
	message := r.FormValue("message")

	// Index existing files by relative path within the directory.
	existing, err := h.queries.ListFilesByDir(ctx, storage.NullText(id))
	if err != nil {
		jsonError(w, "failed to list directory files", http.StatusInternalServerError)
		return
	}
	byRelPath := make(map[string]db.File, len(existing))
	for _, f := range existing {
		rel := strings.TrimPrefix(f.Path, dir.Prefix)
		byRelPath[rel] = f
	}

	paths := r.MultipartForm.Value["path[]"]
	formFiles := r.MultipartForm.File["file"]

	result := dirUpdateResult{Updated: []string{}, Unchanged: []string{}, Added: []string{}}

	for i, fh := range formFiles {
		relPath := fh.Filename
		if i < len(paths) && paths[i] != "" {
			relPath = paths[i]
		}
		relPath = filepath.ToSlash(filepath.Clean(relPath))

		f, err := fh.Open()
		if err != nil {
			log.Printf("update dir: open %s: %v", relPath, err)
			continue
		}

		if existingFile, exists := byRelPath[relPath]; exists {
			res, _, err := h.version.Update(ctx, ver.UpdateParams{
				FileID:     existingFile.ID,
				NewContent: f,
				UploadedBy: uploadedBy,
				Message:    message,
			})
			f.Close()
			if err != nil {
				log.Printf("update dir: update %s: %v", relPath, err)
				continue
			}
			if res == ver.ResultUpdated {
				result.Updated = append(result.Updated, relPath)
			} else {
				result.Unchanged = append(result.Unchanged, relPath)
			}
		} else {
			// New file — add it to the directory.
			err := addFileToDir(ctx, h.queries, h.blobs, dir, relPath, f, uploadedBy)
			f.Close()
			if err != nil {
				log.Printf("update dir: add %s: %v", relPath, err)
				continue
			}
			result.Added = append(result.Added, relPath)
		}
	}

	jsonOK(w, result)
}

// ── GET /files/{id}/versions ─────────────────────────────────────────────────

func (h *versionHandler) handleListVersions(w http.ResponseWriter, r *http.Request) {
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

	versions, err := h.queries.ListVersionsForFile(ctx, id)
	if err != nil {
		jsonError(w, "failed to list versions", http.StatusInternalServerError)
		return
	}

	type versionResponse struct {
		ID            int64  `json:"id"`
		VersionNumber int64  `json:"version_number"`
		Sha256        string `json:"sha256"`
		Size          int64  `json:"size"`
		UploadedBy    string `json:"uploaded_by"`
		Message       string `json:"message"`
		CreatedAt     string `json:"created_at"`
	}

	resp := make([]versionResponse, len(versions))
	for i, v := range versions {
		resp[i] = versionResponse{
			ID:            v.ID,
			VersionNumber: v.VersionNumber,
			Sha256:        v.Sha256,
			Size:          v.Size,
			UploadedBy:    v.UploadedBy,
			Message:       v.Message,
			CreatedAt:     v.CreatedAt,
		}
	}

	jsonOK(w, resp)
}

// addFileToDir creates a new file record inside an existing directory.
// Extracted as a shared helper used by both handleUploadDir and handleUpdateDir.
func addFileToDir(
	ctx context.Context,
	queries *db.Queries,
	blobs *storage.BlobStore,
	dir db.Directory,
	relPath string,
	content interface{ Read([]byte) (int, error) },
	uploadedBy string,
) error {
	fileID := uuid.New().String()

	// content satisfies io.Reader
	type reader interface{ Read([]byte) (int, error) }
	size, sha256hex, err := blobs.Write(fileID, content.(interface {
		Read(p []byte) (n int, err error)
	}))
	if err != nil {
		return err
	}

	fileName := filepath.Base(relPath)
	subDir := filepath.Dir(relPath)
	dirPrefix := ""
	if subDir != "." {
		dirPrefix = subDir + "/"
	}

	_, err = queries.CreateFile(ctx, db.CreateFileParams{
		ID:         fileID,
		Path:       dir.Prefix + relPath,
		FileName:   fileName,
		DirID:      storage.NullText(dir.ID),
		DirPrefix:  dirPrefix,
		Language:   detectLanguage(fileName),
		Size:       size,
		BlobPath:   blobs.BlobPath(fileID),
		Sha256:     sha256hex,
		UploadedBy: uploadedBy,
	})
	if err != nil {
		blobs.Delete(fileID)
		return err
	}
	return nil
}

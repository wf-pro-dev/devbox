package api

import (
	"encoding/json"
	"log"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/wf-pro-dev/devbox/internal/auth"
	"github.com/wf-pro-dev/devbox/internal/db"
	"github.com/wf-pro-dev/devbox/internal/search"
	"github.com/wf-pro-dev/devbox/internal/storage"
	"github.com/wf-pro-dev/devbox/internal/transfer"
	"tailscale.com/client/local"
)

type dirHandler struct {
	queries  *db.Queries
	blobs    *storage.BlobStore
	searcher *search.Searcher
	lc       *local.Client
}

type dirResponse struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Prefix      string         `json:"prefix"`
	Description string         `json:"description"`
	UploadedBy  string         `json:"uploaded_by"`
	CreatedAt   string         `json:"created_at"`
	FileCount   int            `json:"file_count"`
	Files       []fileResponse `json:"files,omitempty"`
}

// ── GET /directories ──────────────────────────────────────────────────────────

func (h *dirHandler) handleListDirs(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	dirs, err := h.queries.ListDirectories(ctx)
	if err != nil {
		jsonError(w, "failed to list directories", http.StatusInternalServerError)
		return
	}

	resp := make([]dirResponse, len(dirs))
	for i, d := range dirs {
		files, _ := h.queries.ListFilesByDir(ctx, storage.NullText(d.ID))
		resp[i] = dirResponse{
			ID:          d.ID,
			Name:        d.Name,
			Prefix:      d.Prefix,
			Description: d.Description,
			UploadedBy:  d.UploadedBy,
			CreatedAt:   d.CreatedAt,
			FileCount:   len(files),
		}
	}
	jsonOK(w, resp)
}

// ── GET /directories/{id} ─────────────────────────────────────────────────────

func (h *dirHandler) handleGetDir(w http.ResponseWriter, r *http.Request) {
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

	files, err := h.queries.ListFilesByDir(ctx, storage.NullText(id))
	if err != nil {
		jsonError(w, "failed to list directory files", http.StatusInternalServerError)
		return
	}

	fileResps := make([]fileResponse, len(files))
	for i, f := range files {
		tags, _ := h.queries.ListTagsForFile(ctx, f.ID)
		names := make([]string, len(tags))
		for j, t := range tags {
			names[j] = t.Name
		}
		fileResps[i] = fileToResponse(f, names)
	}

	jsonOK(w, dirResponse{
		ID:          dir.ID,
		Name:        dir.Name,
		Prefix:      dir.Prefix,
		Description: dir.Description,
		UploadedBy:  dir.UploadedBy,
		CreatedAt:   dir.CreatedAt,
		Files:       fileResps,
		FileCount:   len(files),
	})
}

// ── POST /directories ─────────────────────────────────────────────────────────
// Multipart fields:
//   dir_name    — directory label (required)
//   description — optional
//   file        — repeated, one per file
//   path[]      — repeated, relative path for each file (same order as file[])

func (h *dirHandler) handleUploadDir(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	r.Body = http.MaxBytesReader(w, r.Body, 500<<20)

	if err := r.ParseMultipartForm(50 << 20); err != nil {
		jsonError(w, "request too large or invalid multipart form", http.StatusBadRequest)
		return
	}

	dirName := strings.TrimSpace(r.FormValue("dir_name"))
	if dirName == "" {
		jsonError(w, "dir_name is required", http.StatusBadRequest)
		return
	}

	description := r.FormValue("description")

	uploadedBy := "unknown"
	if id, ok := auth.FromContext(ctx); ok {
		uploadedBy = id.Hostname
	}

	paths := r.MultipartForm.Value["path[]"]
	formFiles := r.MultipartForm.File["file"]
	if len(formFiles) == 0 {
		jsonError(w, "no files provided", http.StatusBadRequest)
		return
	}

	dirID := uuid.New().String()
	prefix := dirName + "/"
	dir, err := h.queries.CreateDirectory(ctx, db.CreateDirectoryParams{
		ID:          dirID,
		Name:        dirName,
		Prefix:      prefix,
		Description: description,
		UploadedBy:  uploadedBy,
	})
	if err != nil {
		jsonError(w, "failed to create directory", http.StatusInternalServerError)
		log.Printf("CreateDirectory: %v", err)
		return
	}

	var uploaded []fileResponse
	for i, fh := range formFiles {
		relPath := fh.Filename
		if i < len(paths) && paths[i] != "" {
			relPath = paths[i]
		}
		relPath = filepath.ToSlash(filepath.Clean(relPath))

		fileName := filepath.Base(relPath)
		subDir := filepath.Dir(relPath)
		dirPrefix := ""
		if subDir != "." {
			dirPrefix = subDir + "/"
		}
		fullPath := prefix + relPath

		f, err := fh.Open()
		if err != nil {
			log.Printf("open form file %s: %v", relPath, err)
			continue
		}

		fileID := uuid.New().String()
		size, sha256hex, err := h.blobs.Write(fileID, f)
		f.Close()
		if err != nil {
			log.Printf("write blob %s: %v", relPath, err)
			continue
		}

		dbFile, err := h.queries.CreateFile(ctx, db.CreateFileParams{
			ID:          fileID,
			Path:        fullPath,
			FileName:    fileName,
			DirID:       storage.NullText(dirID),
			DirPrefix:   dirPrefix,
			Description: "",
			Language:    detectLanguage(fileName),
			Size:        size,
			BlobPath:    h.blobs.BlobPath(fileID),
			Sha256:      sha256hex,
			UploadedBy:  uploadedBy,
		})
		if err != nil {
			log.Printf("CreateFile %s: %v", fullPath, err)
			h.blobs.Delete(fileID)
			continue
		}

		if err := h.searcher.UpdateContent(ctx, fileID, h.blobs); err != nil {
			log.Printf("FTS index %s: %v", fileID, err)
		}

		uploaded = append(uploaded, fileToResponse(dbFile, []string{}))
	}

	w.WriteHeader(http.StatusCreated)
	jsonOK(w, dirResponse{
		ID:          dir.ID,
		Name:        dir.Name,
		Prefix:      dir.Prefix,
		Description: dir.Description,
		UploadedBy:  dir.UploadedBy,
		CreatedAt:   dir.CreatedAt,
		Files:       uploaded,
		FileCount:   len(uploaded),
	})
}

// ── POST /directories/{id}/tags ───────────────────────────────────────────────

func (h *dirHandler) handleTagDir(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := r.PathValue("id")

	if _, err := h.queries.GetDirectory(ctx, id); err != nil {
		if isNotFound(err) {
			jsonError(w, "directory not found", http.StatusNotFound)
			return
		}
		jsonError(w, "failed to get directory", http.StatusInternalServerError)
		return
	}

	var body struct {
		Tags []string `json:"tags"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonError(w, "invalid JSON body", http.StatusBadRequest)
		return
	}

	fileIDs, err := h.queries.ListFileIDsForDir(ctx, storage.NullText(id))
	if err != nil {
		jsonError(w, "failed to list directory files", http.StatusInternalServerError)
		return
	}

	// Upsert tags once, then apply to all files.
	var tagIDs []int64
	for _, name := range body.Tags {
		tag, err := h.queries.CreateTag(ctx, name)
		if err != nil {
			log.Printf("CreateTag %q: %v", name, err)
			continue
		}
		tagIDs = append(tagIDs, tag.ID)
	}

	for _, fileID := range fileIDs {
		for _, tagID := range tagIDs {
			if err := h.queries.AddTagToFile(ctx, db.AddTagToFileParams{FileID: fileID, TagID: tagID}); err != nil {
				log.Printf("AddTagToFile %s: %v", fileID, err)
			}
		}
	}

	jsonOK(w, map[string]interface{}{
		"tagged_files": len(fileIDs),
		"tags":         body.Tags,
	})
}

// ── POST /directories/{id}/deliver ───────────────────────────────────────────

func (h *dirHandler) handleDeliverDir(w http.ResponseWriter, r *http.Request) {
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

	var req struct {
		Targets   []string `json:"targets"`
		Broadcast bool     `json:"broadcast"`
		DestDir   string   `json:"dest_dir"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid JSON body", http.StatusBadRequest)
		return
	}
	if req.DestDir == "" {
		req.DestDir = "~/devbox-received/" + dir.Name
	}

	targets := req.Targets
	if req.Broadcast {
		status, err := h.lc.Status(ctx)
		if err != nil {
			jsonError(w, "failed to list peers", http.StatusInternalServerError)
			return
		}
		for _, p := range status.Peer {
			if !p.Online {
				continue
			}
			dns := strings.TrimSuffix(p.DNSName, ".")
			if parts := strings.SplitN(dns, ".", 2); len(parts) > 0 {
				targets = append(targets, parts[0])
			}
		}
	}

	files, err := h.queries.ListFilesByDir(ctx, storage.NullText(id))
	if err != nil {
		jsonError(w, "failed to list directory files", http.StatusInternalServerError)
		return
	}

	type fileResult struct {
		Path    string            `json:"path"`
		Results []transfer.Result `json:"results"`
	}

	var allResults []fileResult
	for _, f := range files {
		destDir := req.DestDir
		if f.DirPrefix != "" {
			destDir = req.DestDir + "/" + strings.TrimSuffix(f.DirPrefix, "/")
		}
		results := transfer.Deliver(ctx, h.lc, transfer.Delivery{
			FileID:   f.ID,
			FileName: f.FileName,
			BlobPath: h.blobs.BlobPath(f.ID),
			Targets:  targets,
			DestDir:  destDir,
		})
		allResults = append(allResults, fileResult{Path: f.Path, Results: results})
	}

	jsonOK(w, allResults)
}

// ── DELETE /directories/{id} ──────────────────────────────────────────────────

func (h *dirHandler) handleDeleteDir(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := r.PathValue("id")

	files, _ := h.queries.ListFilesByDir(ctx, storage.NullText(id))
	for _, f := range files {
		if err := h.blobs.Delete(f.ID); err != nil {
			log.Printf("delete blob %s: %v", f.ID, err)
		}
	}

	if err := h.queries.DeleteDirectory(ctx, id); err != nil {
		if isNotFound(err) {
			jsonError(w, "directory not found", http.StatusNotFound)
			return
		}
		jsonError(w, "failed to delete directory", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

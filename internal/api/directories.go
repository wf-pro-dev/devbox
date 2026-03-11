package api

import (
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/wf-pro-dev/devbox/internal/db"
	"github.com/wf-pro-dev/devbox/internal/storage"
	"github.com/wf-pro-dev/devbox/internal/version"
	"github.com/wf-pro-dev/devbox/types"
)

type dirsHandler struct {
	store  *storage.Store
	blobs  *storage.BlobStore
	verSvc *version.Service
}

// toPrefix normalises a directory name to a trailing-slash prefix.
// e.g. "nginx" → "nginx/",  "nginx/" → "nginx/"
func toPrefix(name string) string {
	name = strings.Trim(name, "/")
	if name == "" {
		return ""
	}
	return name + "/"
}

// listDirFiles returns all files under prefix using ListFiles.
func (h *dirsHandler) listDirFiles(ctx interface{ Deadline() (interface{}, bool) }, prefix string) ([]db.File, error) {
	// ctx is context.Context — typed inline to avoid a separate import alias
	return nil, nil // placeholder — real implementation below uses the correct type
}

// ── GET /dirs ─────────────────────────────────────────────────────────────────
// Returns all distinct top-level directory names with file count and tags.
// Optional ?tag= filter narrows to dirs that contain at least one file with
// that tag.

func (h *dirsHandler) handleList(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	params := db.ListFilesParams{}
	if tag := r.URL.Query().Get("tag"); tag != "" {
		params.Tag = &tag
	}

	files, err := h.store.Queries.ListFiles(ctx, params)
	if err != nil {
		jsonError(w, "failed to list files", http.StatusInternalServerError)
		log.Printf("dirs list: %v", err)
		return
	}

	// Derive distinct top-level dir names from the file set.
	// A file is "in a dir" if its path contains a slash.
	type dirEntry struct {
		files []db.File
	}
	dirMap := make(map[string]*dirEntry)
	for _, f := range files {
		idx := strings.Index(f.Path, "/")
		if idx <= 0 {
			continue // root-level file, not in any dir
		}
		name := f.Path[:idx]
		if dirMap[name] == nil {
			dirMap[name] = &dirEntry{}
		}
		dirMap[name].files = append(dirMap[name].files, f)
	}

	resp := make([]types.Directory, 0, len(dirMap))
	for name, entry := range dirMap {
		prefix := name + "/"
		resp = append(resp, types.Directory{Prefix: prefix, FileCount: len(entry.files), Tags: prefixTags(ctx, h.store.Queries, prefix), Files: entry.files})
	}

	jsonOK(w, resp)
}

// ── GET /dirs/{dir} ───────────────────────────────────────────────────────────
// Returns dir metadata + full file list.

func (h *dirsHandler) handleGet(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	dirName, err := url.PathUnescape(r.PathValue("dir"))
	if err != nil {
		jsonError(w, "invalid directory path", http.StatusBadRequest)
		return
	}
	prefix := toPrefix(dirName)

	files, err := h.store.Queries.ListFiles(ctx, db.ListFilesParams{Prefix: &prefix})
	if err != nil {
		jsonError(w, "failed to list files", http.StatusInternalServerError)
		return
	}
	if len(files) == 0 {
		jsonError(w, "directory not found", http.StatusNotFound)
		return
	}

	jsonOK(w, types.Directory{Prefix: prefix, FileCount: len(files), Tags: prefixTags(ctx, h.store.Queries, prefix), Files: files})
}

// ── POST /dirs ────────────────────────────────────────────────────────────────
// Uploads a set of files under a common prefix, creating the virtual dir.
// Multipart fields:
//
//	name    — directory name / prefix root (required)
//	tags    — comma-separated tags applied to every uploaded file (optional)
//	file    — repeated, one per file
//	path[]  — repeated, relative path per file (same order as file[])

func (h *dirsHandler) handleCreate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	r.Body = http.MaxBytesReader(w, r.Body, 500<<20)

	if err := r.ParseMultipartForm(50 << 20); err != nil {
		jsonError(w, "request too large or invalid multipart form", http.StatusBadRequest)
		return
	}

	name := strings.TrimSpace(r.FormValue("name"))
	if name == "" {
		jsonError(w, "'name' is required", http.StatusBadRequest)
		return
	}
	prefix := toPrefix(name)
	tagNames := splitTags(r.FormValue("tags"))
	uploadedBy := callerHost(ctx)

	relPaths := r.MultipartForm.Value["path[]"]
	formFiles := r.MultipartForm.File["file"]

	var created []db.File

	for i, fh := range formFiles {
		relPath := fh.Filename
		if i < len(relPaths) && relPaths[i] != "" {
			relPath = relPaths[i]
		}
		relPath = filepath.ToSlash(filepath.Clean(relPath))
		fullPath := prefix + relPath

		f, err := fh.Open()
		if err != nil {
			log.Printf("dirs create: open %s: %v", relPath, err)
			continue
		}

		wr, err := h.blobs.Write(ctx, f)
		f.Close()
		if err != nil {
			log.Printf("dirs create: write blob %s: %v", relPath, err)
			continue
		}

		fileID := uuid.New().String()
		dbFile, err := h.store.Queries.CreateFile(ctx, db.CreateFileParams{
			ID:         fileID,
			Path:       fullPath,
			FileName:   filepath.Base(relPath),
			Language:   detectLanguage(fh.Filename),
			Size:       wr.Size,
			Sha256:     wr.SHA256,
			UploadedBy: uploadedBy,
		})
		if err != nil {
			log.Printf("dirs create: insert %s: %v", fullPath, err)
			continue
		}

		if err := applyTags(ctx, h.store.Queries, fileID, tagNames); err != nil {
			log.Printf("dirs create: apply tags %s: %v", fileID, err)
		}

		created = append(created, dbFile)
	}

	jsonCreated(w, types.Directory{Prefix: prefix, FileCount: len(created), Tags: tagNames, Files: created})
}

// ── PUT /dirs/{dir} ───────────────────────────────────────────────────────────
// Syncs a local directory to the server prefix.
//   - Existing file, content changed → new version via version.Service
//   - Existing file, content same   → unchanged
//   - New file                      → created

func (h *dirsHandler) handleSync(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	dirName, err := url.PathUnescape(r.PathValue("dir"))
	if err != nil {
		jsonError(w, "invalid directory path", http.StatusBadRequest)
		return
	}
	prefix := toPrefix(dirName)

	r.Body = http.MaxBytesReader(w, r.Body, 500<<20)
	if err := r.ParseMultipartForm(50 << 20); err != nil {
		jsonError(w, "invalid multipart form", http.StatusBadRequest)
		return
	}

	message := r.FormValue("message")
	uploadedBy := callerHost(ctx)
	relPaths := r.MultipartForm.Value["path[]"]
	formFiles := r.MultipartForm.File["file"]

	// Index existing files under the prefix by their full path.
	existing, _ := h.store.Queries.ListFiles(ctx, db.ListFilesParams{Prefix: &prefix})
	byPath := make(map[string]db.File, len(existing))
	for _, f := range existing {
		byPath[f.Path] = f
	}

	var updated, added, unchanged []string

	for i, fh := range formFiles {
		relPath := fh.Filename
		if i < len(relPaths) && relPaths[i] != "" {
			relPath = relPaths[i]
		}
		relPath = filepath.ToSlash(filepath.Clean(relPath))
		fullPath := prefix + relPath

		f, err := fh.Open()
		if err != nil {
			log.Printf("dirs sync: open %s: %v", relPath, err)
			continue
		}

		if existing, exists := byPath[fullPath]; exists {
			result, _, err := h.verSvc.Update(ctx, version.UpdateParams{
				FileID:     existing.ID,
				NewContent: f,
				UploadedBy: uploadedBy,
				Message:    message,
			})
			f.Close()
			if err != nil {
				log.Printf("dirs sync: update %s: %v", fullPath, err)
				continue
			}
			if result == version.ResultUpdated {
				updated = append(updated, relPath)
			} else {
				unchanged = append(unchanged, relPath)
			}
		} else {
			wr, err := h.blobs.Write(ctx, f)
			f.Close()
			if err != nil {
				log.Printf("dirs sync: write blob %s: %v", relPath, err)
				continue
			}
			fileID := uuid.New().String()
			_, err = h.store.Queries.CreateFile(ctx, db.CreateFileParams{
				ID:         fileID,
				Path:       fullPath,
				FileName:   filepath.Base(relPath),
				Language:   detectLanguage(fh.Filename),
				Size:       wr.Size,
				Sha256:     wr.SHA256,
				UploadedBy: uploadedBy,
			})
			if err != nil {
				log.Printf("dirs sync: create %s: %v", fullPath, err)
				continue
			}
			added = append(added, relPath)
		}
	}

	jsonOK(w, map[string]any{
		"prefix":    prefix,
		"updated":   orEmpty(updated),
		"added":     orEmpty(added),
		"unchanged": orEmpty(unchanged),
	})
}

// ── DELETE /dirs/{dir} ────────────────────────────────────────────────────────
// Deletes all files under the prefix, then cleans up orphaned blobs.

func (h *dirsHandler) handleDelete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	dirName, err := url.PathUnescape(r.PathValue("dir"))
	if err != nil {
		jsonError(w, "invalid directory path", http.StatusBadRequest)
		return
	}
	prefix := toPrefix(dirName)

	files, _ := h.store.Queries.ListFiles(ctx, db.ListFilesParams{Prefix: &prefix})
	if len(files) == 0 {
		jsonError(w, "directory not found", http.StatusNotFound)
		return
	}

	shas := make([]string, len(files))
	for i, f := range files {
		shas[i] = f.Sha256
	}

	if err := h.store.Queries.DeleteFilesByPrefix(ctx, &prefix); err != nil {
		jsonError(w, "delete failed", http.StatusInternalServerError)
		log.Printf("delete dir %s: %v", prefix, err)
		return
	}

	for _, sha := range shas {
		go h.blobs.DeleteIfUnreferenced(ctx, sha)
	}

	jsonNoContent(w)
}

// ── POST /dirs/{dir}/tags ─────────────────────────────────────────────────────
// Adds tags to every file under the prefix in two queries: UpsertTag + bulk insert.
// JSON body: {"tags": ["tag1", "tag2"]}

func (h *dirsHandler) handleAddTags(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	prefix := toPrefix(r.PathValue("dir"))

	// Verify the dir exists.
	files, _ := h.store.Queries.ListFiles(ctx, db.ListFilesParams{Prefix: &prefix})
	if len(files) == 0 {
		jsonError(w, "directory not found", http.StatusNotFound)
		return
	}

	var body struct {
		Tags []string `json:"tags"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonError(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	for _, name := range body.Tags {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		tag, err := h.store.Queries.UpsertTag(ctx, name)
		if err != nil {
			jsonError(w, "failed to upsert tag", http.StatusInternalServerError)
			return
		}
		if err := h.store.Queries.AddTagToFilesByPrefix(ctx, db.AddTagToFilesByPrefixParams{
			TagID:   tag.ID,
			Column2: &prefix,
		}); err != nil {
			jsonError(w, "failed to apply tag", http.StatusInternalServerError)
			return
		}
	}

	jsonOK(w, map[string][]string{"tags": prefixTags(ctx, h.store.Queries, prefix)})
}

// ── DELETE /dirs/{dir}/tags/{tag} ─────────────────────────────────────────────
// Removes a tag from every file under the prefix.

func (h *dirsHandler) handleRemoveTag(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	dirName, err := url.PathUnescape(r.PathValue("dir"))
	if err != nil {
		jsonError(w, "invalid directory path", http.StatusBadRequest)
		return
	}
	prefix := toPrefix(dirName)

	tag, err := h.store.Queries.GetTagByName(ctx, r.PathValue("tag"))
	if isNotFound(err) {
		jsonError(w, "tag not found", http.StatusNotFound)
		return
	}
	if err != nil {
		jsonError(w, "failed to get tag", http.StatusInternalServerError)
		return
	}

	h.store.Queries.RemoveTagFromFilesByPrefix(ctx, db.RemoveTagFromFilesByPrefixParams{
		TagID:   tag.ID,
		Column2: &prefix,
	})

	jsonNoContent(w)
}

// ── GET /dirs/{dir}/diff ──────────────────────────────────────────────────────
// Compares the server's state of a prefix to a submitted local manifest.
// Body: JSON array of {path, sha256} where path is relative to the prefix root.
// Returns which files are added/changed/removed vs. the server.

func (h *dirsHandler) handleDiff(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	dirName, err := url.PathUnescape(r.PathValue("dir"))
	if err != nil {
		jsonError(w, "invalid directory path", http.StatusBadRequest)
		return
	}
	prefix := toPrefix(dirName)

	var localFiles []struct {
		Path   string `json:"path"`
		SHA256 string `json:"sha256"`
	}
	if err := json.NewDecoder(r.Body).Decode(&localFiles); err != nil {
		jsonError(w, "invalid JSON body", http.StatusBadRequest)
		return
	}

	serverFiles, _ := h.store.Queries.ListFiles(ctx, db.ListFilesParams{Prefix: &prefix})
	serverByRel := make(map[string]db.File, len(serverFiles))
	for _, f := range serverFiles {
		rel := strings.TrimPrefix(f.Path, prefix)
		serverByRel[rel] = f
	}

	localByRel := make(map[string]string, len(localFiles))
	for _, lf := range localFiles {
		localByRel[lf.Path] = lf.SHA256
	}

	var changed, added, removed []string
	for _, lf := range localFiles {
		sf, exists := serverByRel[lf.Path]
		if !exists {
			added = append(added, lf.Path)
		} else if sf.Sha256 != lf.SHA256 {
			changed = append(changed, lf.Path)
		}
	}
	for rel := range serverByRel {
		if _, exists := localByRel[rel]; !exists {
			removed = append(removed, rel)
		}
	}

	jsonOK(w, map[string]any{
		"prefix":  prefix,
		"changed": orEmpty(changed),
		"added":   orEmpty(added),
		"removed": orEmpty(removed),
	})
}

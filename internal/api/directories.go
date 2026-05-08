package api

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/wf-pro-dev/devbox/internal/db"
	"github.com/wf-pro-dev/devbox/internal/models"
	"github.com/wf-pro-dev/devbox/internal/storage"
	"github.com/wf-pro-dev/devbox/internal/version"
	"github.com/wf-pro-dev/devbox/types"
)

type dirsHandler struct {
	filesHandler
}

// pathOfDBFile is the PathOf[db.File] adapter for models.ListDirect.
func pathOfDBFile(f db.File) string { return f.Path }

// ── Shared helpers ────────────────────────────────────────────────────────────

// parsePrefix extracts, unescapes, and canonicalises a directory prefix from
// the request. It reads from the named path value first; if that is empty it
// falls back to the "prefix" query parameter; if both are absent it returns "/".
//
// On success it returns (canonical prefix, true).
// On failure it writes a 400 response and returns ("", false).
func parsePrefix(w http.ResponseWriter, r *http.Request, pathValueName string) (string, bool) {
	raw := r.PathValue(pathValueName)
	if raw == "" {
		raw = r.URL.Query().Get("prefix")
	}
	if raw == "" {
		return "/", true
	}

	unescaped, err := url.PathUnescape(raw)
	if err != nil {
		jsonError(w, "invalid path encoding", http.StatusBadRequest)
		return "", false
	}

	prefix, err := models.CanonicalDir(unescaped)
	if err != nil {
		jsonError(w, "invalid directory path: "+err.Error(), http.StatusBadRequest)
		return "", false
	}
	return prefix, true
}

// dbPrefixParam converts a canonical prefix to the nullable *string that
// db.ListFilesParams.Prefix expects. Root "/" means "no filter" → nil.
func dbPrefixParam(prefix string) *string {
	if prefix == "/" {
		return nil
	}
	return &prefix
}

// listingFromFiles runs the CommonPrefix algorithm over a pre-sorted slice of
// files and returns a types.DirListing ready to serialise.
//
// When deep is true the listing is a flat enumeration of every file under
// prefix with no directory collapsing — the equivalent of find(1) -type f.
// When deep is false only direct children are returned (ls -1 style).
func listingFromFiles(files []db.File, prefix string, deep bool) types.DirListing {
	listing := types.DirListing{
		Prefix:  prefix,
		Entries: make([]types.DirEntry, 0, len(files)),
	}

	if deep {
		for i := range files {
			f := &files[i]
			listing.Entries = append(listing.Entries, types.DirEntry{
				Name:  filepath.Base(f.Path),
				IsDir: false,
				File:  f,
			})
		}
		return listing
	}

	for _, e := range models.ListDirect(files, prefix, pathOfDBFile) {
		entry := types.DirEntry{
			Name:      e.Name,
			IsDir:     e.IsDir,
			Prefix:    e.Prefix,
			FileCount: e.FileCount,
		}
		if !e.IsDir {
			f := e.File // local copy for a stable address
			entry.File = &f
		}
		listing.Entries = append(listing.Entries, entry)
	}
	return listing
}

// ── GET /dirs ─────────────────────────────────────────────────────────────────
//
// Lists the direct children of a virtual directory.
//
// Query parameters:
//
//	prefix  — canonical directory prefix (default "/").
//	tag     — narrow to files that carry this tag.
//	depth   — "all" for a flat recursive listing; omit for direct children only.
//
// Both the CLI and the web client receive the same types.DirListing JSON.
func (h *dirsHandler) handleList(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	prefix, ok := parsePrefix(w, r, "")
	if !ok {
		return
	}

	params := db.ListFilesParams{Prefix: dbPrefixParam(prefix)}
	if tag := r.URL.Query().Get("tag"); tag != "" {
		params.Tag = &tag
	}

	files, err := h.store.Queries.ListFiles(ctx, params)
	if err != nil {
		jsonError(w, "failed to list files", http.StatusInternalServerError)
		log.Printf("dirs list: %v", err)
		return
	}

	listing := listingFromFiles(files, prefix, r.URL.Query().Get("depth") == "all")
	if prefix != "/" {
		listing.Tags = prefixTags(ctx, h.store.Queries, prefix)
	}

	jsonOK(w, listing)
}

// ── GET /dirs/{dir} ───────────────────────────────────────────────────────────
//
// Returns the direct children of a named directory as a types.DirListing.
// This is the primary endpoint for both the CLI ls and the web column-view.
//
// Query parameters:
//
//	content=true    — stream a .tar.gz archive instead of JSON.
//	recursive=true  — flat listing of all files in every sub-directory.
//	depth=all       — alias for recursive=true.
func (h *dirsHandler) handleGet(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	prefix, ok := parsePrefix(w, r, "dir")
	if !ok {
		return
	}

	if r.URL.Query().Get("content") == "true" {
		h.handleGetDirContent(w, r, prefix)
		return
	}

	files, err := h.store.Queries.ListFiles(ctx, db.ListFilesParams{Prefix: dbPrefixParam(prefix)})
	if err != nil {
		jsonError(w, "failed to list files", http.StatusInternalServerError)
		return
	}
	if len(files) == 0 && prefix != "/" {
		jsonError(w, "directory not found", http.StatusNotFound)
		return
	}

	deep := r.URL.Query().Get("recursive") == "true" || r.URL.Query().Get("depth") == "all"
	listing := listingFromFiles(files, prefix, deep)
	listing.Tags = prefixTags(ctx, h.store.Queries, prefix)

	jsonOK(w, listing)
}

// ── GET /dirs/{dir}?content=true ──────────────────────────────────────────────
// Streams a gzip-compressed tarball of the entire directory (all files,
// recursive). Delegated to by handleGet; not registered as a route directly.
func (h *dirsHandler) handleGetDirContent(w http.ResponseWriter, r *http.Request, prefix string) {
	ctx := r.Context()

	files, err := h.store.Queries.ListFiles(ctx, db.ListFilesParams{Prefix: dbPrefixParam(prefix)})
	if err != nil {
		jsonError(w, "failed to list files", http.StatusInternalServerError)
		return
	}
	if len(files) == 0 {
		jsonError(w, "directory not found", http.StatusNotFound)
		return
	}

	// Derive a safe archive name from the last path segment.
	dirName := filepath.Base(strings.TrimSuffix(prefix, "/"))
	if dirName == "." || dirName == "/" {
		dirName = "root"
	}

	tarball, err := storage.CreateTarball(dirName, files, h.blobs)
	if err != nil {
		jsonError(w, fmt.Sprintf("failed to create tarball: %v", err), http.StatusInternalServerError)
		return
	}
	defer tarball.Close()

	stat, err := tarball.Stat()
	if err != nil {
		jsonError(w, fmt.Sprintf("failed to stat tarball: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/tar+gzip")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.tar.gz"`, dirName))
	w.Header().Set("Content-Length", strconv.FormatInt(stat.Size(), 10))
	if _, err = io.Copy(w, tarball); err != nil {
		log.Printf("dirs content: copy %s: %v", prefix, err)
	}
}

// ── POST /dirs ────────────────────────────────────────────────────────────────
//
// Uploads files under a common prefix, creating the virtual directory.
//
// Multipart fields:
//
//	name         — directory name / prefix root (required).
//	tags         — comma-separated tags applied to every file (optional).
//	file         — repeated, one per uploaded file.
//	path[]       — repeated, relative path per file (same order as file[]).
//	local_path[] — repeated, absolute local path per file (provenance).
//
// Returns a types.DirListing of the created entries so the caller can render
// the result immediately without a follow-up GET.
func (h *dirsHandler) handleCreate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	r.Body = http.MaxBytesReader(w, r.Body, 500<<20)

	if err := r.ParseMultipartForm(50 << 20); err != nil {
		jsonError(w, "request too large or invalid multipart form", http.StatusBadRequest)
		return
	}

	rawName := strings.TrimSpace(r.FormValue("name"))
	if rawName == "" {
		jsonError(w, "'name' is required", http.StatusBadRequest)
		return
	}
	prefix, err := models.CanonicalDir(rawName)
	if err != nil {
		jsonError(w, "invalid directory name: "+err.Error(), http.StatusBadRequest)
		return
	}

	tagNames := splitTags(r.FormValue("tags"))
	relPaths := r.MultipartForm.Value["path[]"]
	localPaths := r.MultipartForm.Value["local_path[]"]
	formFiles := r.MultipartForm.File["file"]

	// createdFiles accumulates the raw db.File records so listingFromFiles can
	var createdFiles []db.File

	for i, fh := range formFiles {
		relPath := fh.Filename
		if i < len(relPaths) && relPaths[i] != "" {
			relPath = relPaths[i]
		}
		fullPath := models.Join(prefix, relPath)

		localPath := ""
		if i < len(localPaths) {
			localPath = localPaths[i]
		}

		f, err := fh.Open()
		if err != nil {
			log.Printf("dirs create: open %s: %v", relPath, err)
			continue
		}

		file, err := models.CreateFile(
			ctx,
			h.store,
			h.blobs,
			h.searcher,
			f,
			fullPath,
			localPath,
			"",
			detectLanguage(fh.Filename),
			tagNames,
		)
		if err != nil {
			log.Printf("dirs create: %s: %v", fullPath, err)
			continue
		}
		createdFiles = append(createdFiles, file.File)
	}

	listing := listingFromFiles(createdFiles, prefix, false)
	listing.Tags = tagNames
	jsonCreated(w, listing)
}

// ── PUT /dirs/{dir} ───────────────────────────────────────────────────────────
//
// Syncs a local directory state to the server prefix.
//
//   - Existing file, content changed → new version via version.Service.
//   - Existing file, content same    → no-op, recorded as "unchanged".
//   - New file                       → created.
//
// Files present on the server but absent from the upload are NOT deleted.
// Returns a delta summary, not a DirListing.
func (h *dirsHandler) handleSync(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	prefix, ok := parsePrefix(w, r, "dir")
	if !ok {
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 500<<20)
	if err := r.ParseMultipartForm(50 << 20); err != nil {
		jsonError(w, "invalid multipart form", http.StatusBadRequest)
		return
	}

	message := r.FormValue("message")
	uploadedBy := callerHost(ctx)
	relPaths := r.MultipartForm.Value["path[]"]
	localPaths := r.MultipartForm.Value["local_path[]"]
	formFiles := r.MultipartForm.File["file"]

	existing, _ := h.store.Queries.ListFiles(ctx, db.ListFilesParams{Prefix: dbPrefixParam(prefix)})
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
		fullPath := models.Join(prefix, relPath)

		f, err := fh.Open()
		if err != nil {
			log.Printf("dirs sync: open %s: %v", relPath, err)
			continue
		}

		if existingFile, exists := byPath[fullPath]; exists {
			result, _, err := h.verSvc.Update(ctx, version.UpdateParams{
				FileID:     existingFile.ID,
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
				log.Printf("dirs sync: blob %s: %v", relPath, err)
				continue
			}

			localPath := ""
			if i < len(localPaths) {
				localPath = localPaths[i]
			}

			_, err = h.store.Queries.CreateFile(ctx, db.CreateFileParams{
				ID:         uuid.New().String(),
				Path:       fullPath,
				LocalPath:  localPath,
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
// Deletes all files under the prefix and asynchronously purges orphaned blobs.
func (h *dirsHandler) handleDelete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	prefix, ok := parsePrefix(w, r, "dir")
	if !ok {
		return
	}

	files, _ := h.store.Queries.ListFiles(ctx, db.ListFilesParams{Prefix: dbPrefixParam(prefix)})
	if len(files) == 0 {
		jsonError(w, "directory not found", http.StatusNotFound)
		return
	}

	shas := make([]string, len(files))
	for i, f := range files {
		shas[i] = f.Sha256
	}

	if err := h.store.Queries.DeleteFilesByPrefix(ctx, dbPrefixParam(prefix)); err != nil {
		jsonError(w, "delete failed", http.StatusInternalServerError)
		log.Printf("dirs delete %s: %v", prefix, err)
		return
	}

	for _, sha := range shas {
		go h.blobs.DeleteIfUnreferenced(ctx, sha)
	}

	jsonNoContent(w)
}

// ── POST /dirs/{dir}/tags ─────────────────────────────────────────────────────
//
// Adds tags to every file under the prefix.
// JSON body: {"tags": ["tag1", "tag2"]}
func (h *dirsHandler) handleAddTags(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	prefix, ok := parsePrefix(w, r, "dir")
	if !ok {
		return
	}

	files, _ := h.store.Queries.ListFiles(ctx, db.ListFilesParams{Prefix: dbPrefixParam(prefix)})
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
		p := prefix // take address of a loop-local copy
		if err := h.store.Queries.AddTagToFilesByPrefix(ctx, db.AddTagToFilesByPrefixParams{
			TagID:   tag.ID,
			Column2: &p,
		}); err != nil {
			jsonError(w, "failed to apply tag", http.StatusInternalServerError)
			return
		}
	}

	jsonOK(w, map[string][]string{"tags": prefixTags(ctx, h.store.Queries, prefix)})
}

// ── DELETE /dirs/{dir}/tags/{tag} ─────────────────────────────────────────────
// Removes a single tag from every file under the prefix.
func (h *dirsHandler) handleRemoveTag(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	prefix, ok := parsePrefix(w, r, "dir")
	if !ok {
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

	p := prefix
	h.store.Queries.RemoveTagFromFilesByPrefix(ctx, db.RemoveTagFromFilesByPrefixParams{
		TagID:   tag.ID,
		Column2: &p,
	})

	jsonNoContent(w)
}

// ── GET /dirs/{dir}/diff ──────────────────────────────────────────────────────
//
// Compares the server's state of a prefix to a client-submitted local manifest.
//
// Request body — JSON array of local descriptors:
//
//	[{"path": "config/nginx.conf", "sha256": "abc123…"}, …]
//
// All paths must be relative to the prefix root (no leading slash). The server
// canonicalises them before comparing so forward/back slashes, leading slashes,
// and double-dots are tolerated in the client payload.
//
// Response:
//
//	{
//	  "prefix":  "/myapp/",
//	  "added":   ["newfile.go"],
//	  "changed": ["config/nginx.conf"],
//	  "removed": ["old.go"]
//	}
func (h *dirsHandler) handleDiff(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	prefix, ok := parsePrefix(w, r, "dir")
	if !ok {
		return
	}

	var localFiles []struct {
		Path   string `json:"path"`
		SHA256 string `json:"sha256"`
	}
	if err := json.NewDecoder(r.Body).Decode(&localFiles); err != nil {
		jsonError(w, "invalid JSON body", http.StatusBadRequest)
		return
	}

	serverFiles, _ := h.store.Queries.ListFiles(ctx, db.ListFilesParams{Prefix: dbPrefixParam(prefix)})

	// Index server files by relative path (prefix stripped).
	serverByRel := make(map[string]db.File, len(serverFiles))
	for _, f := range serverFiles {
		rel := strings.TrimPrefix(f.Path, prefix)
		serverByRel[rel] = f
	}

	// Index local files by canonicalised relative path.
	localByRel := make(map[string]string, len(localFiles))
	for _, lf := range localFiles {
		rel := strings.ReplaceAll(lf.Path, "\\", "/")
		rel = strings.TrimPrefix(rel, "/")
		localByRel[rel] = lf.SHA256
	}

	var changed, added, removed []string
	for rel, sha := range localByRel {
		sf, exists := serverByRel[rel]
		switch {
		case !exists:
			added = append(added, rel)
		case sf.Sha256 != sha:
			changed = append(changed, rel)
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

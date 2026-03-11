package api

import (
	"encoding/json"
	"net/http"
	"os"
	"strconv"

	"github.com/wf-pro-dev/devbox/internal/auth"
	"github.com/wf-pro-dev/devbox/internal/search"
	"github.com/wf-pro-dev/devbox/internal/storage"
	"github.com/wf-pro-dev/devbox/internal/version"
	"tailscale.com/client/local"
)

// NewRouter wires all handlers and returns the root http.Handler.
func NewRouter(lc *local.Client, store *storage.Store, blobs *storage.BlobStore) http.Handler {
	maxVersions := 10
	if v := os.Getenv("DEVBOX_MAX_VERSIONS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			maxVersions = n
		}
	}

	searcher := search.New(store.DB)
	verSvc := version.New(store.Queries, blobs, maxVersions)

	fh := &filesHandler{
		store:    store,
		blobs:    blobs,
		searcher: searcher,
		verSvc:   verSvc,
	}
	dh := &dirsHandler{
		store:  store,
		blobs:  blobs,
		verSvc: verSvc,
	}
	sh := &sendHandler{
		store: store,
		blobs: blobs,
		lc:    lc,
	}
	sch := &searchHandler{
		searcher: searcher,
	}

	mux := http.NewServeMux()

	// ── Health ───────────────────────────────────────────────────────────────
	mux.HandleFunc("GET /health", handleHealth)

	// ── Files ────────────────────────────────────────────────────────────────
	// GET  /files              list — optional ?dir= ?tag= ?lang= ?q= filters
	// POST /files              upload a single file
	// GET  /files/{id}         download blob (?meta=true for JSON metadata)
	// PUT  /files/{id}         replace content, snapshot old version
	// PATCH /files/{id}        edit description / language / path
	// DELETE /files/{id}       delete file and clean up blob
	// POST /files/{id}/tags              add tags
	// DELETE /files/{id}/tags/{tag}      remove one tag
	// POST /files/{id}/copy              copy to new path (same blob)
	// POST /files/{id}/move              rename/move to new path
	// GET  /files/{id}/versions          list version history
	// POST /files/{id}/versions/{n}/rollback  restore version n
	// GET  /files/{id}/diff              metadata diff between two versions
	// POST /files/{id}/deliver           push to tailnet peers
	mux.HandleFunc("GET /files", fh.handleList)
	mux.HandleFunc("POST /files", fh.handleUpload)
	mux.HandleFunc("GET /files/{id}", fh.handleGet)
	mux.HandleFunc("PUT /files/{id}", fh.handleUpdate)
	mux.HandleFunc("PATCH /files/{id}", fh.handleEditMeta)
	mux.HandleFunc("DELETE /files/{id}", fh.handleDelete)
	mux.HandleFunc("POST /files/{id}/tags", fh.handleAddTags)
	mux.HandleFunc("DELETE /files/{id}/tags/{tag}", fh.handleRemoveTag)
	mux.HandleFunc("POST /files/{id}/copy", fh.handleCopy)
	mux.HandleFunc("POST /files/{id}/move", fh.handleMove)
	mux.HandleFunc("GET /files/{id}/versions", fh.handleListVersions)
	mux.HandleFunc("GET /files/{id}/versions/{n}", fh.handleGetVersion)
	mux.HandleFunc("POST /files/{id}/versions/{n}/rollback", fh.handleRollback)
	mux.HandleFunc("GET /files/{id}/diff", fh.handleDiff)
	mux.HandleFunc("POST /files/{id}/send", sh.handleSendFile)

	// ── Dirs (virtual — no DB entity, resolved from path prefix) ─────────────
	// GET  /dirs               list all dirs (?tag= filter)
	// POST /dirs               create dir by uploading initial files
	// GET  /dirs/{dir}         dir metadata + file list
	// PUT  /dirs/{dir}         sync local dir (add/update files)
	// DELETE /dirs/{dir}       delete all files under the prefix
	// POST /dirs/{dir}/tags              add tags to all files in dir
	// DELETE /dirs/{dir}/tags/{tag}      remove tag from all files in dir
	// GET  /dirs/{dir}/diff              compare local manifest to server
	// POST /dirs/{dir}/deliver           push all files in dir to tailnet peers
	//
	// NOTE: PATCH /dirs/{dir} is intentionally absent — a virtual dir has no
	// metadata row to update. Rename a dir by moving individual files via
	// PATCH /files/{id} with a new path.
	mux.HandleFunc("GET /dirs", dh.handleList)
	mux.HandleFunc("POST /dirs", dh.handleCreate)
	mux.HandleFunc("GET /dirs/{dir}", dh.handleGet)
	mux.HandleFunc("PUT /dirs/{dir}", dh.handleSync)
	mux.HandleFunc("DELETE /dirs/{dir}", dh.handleDelete)
	mux.HandleFunc("POST /dirs/{dir}/tags", dh.handleAddTags)
	mux.HandleFunc("DELETE /dirs/{dir}/tags/{tag}", dh.handleRemoveTag)
	mux.HandleFunc("GET /dirs/{dir}/diff", dh.handleDiff)
	mux.HandleFunc("POST /dirs/{dir}/send", sh.handleSendDir)

	// ── Search ───────────────────────────────────────────────────────────────
	mux.HandleFunc("GET /search", sch.handleSearch)

	// ── Peers ────────────────────────────────────────────────────────────────
	mux.HandleFunc("GET /peers", sh.handleListPeers)

	var h http.Handler = mux
	h = auth.Middleware(lc)(h)
	return h
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	resp := map[string]string{"status": "ok", "service": "devbox"}
	if id, ok := auth.FromContext(r.Context()); ok {
		resp["caller_host"] = id.Hostname
		resp["caller_user"] = id.UserLogin
		resp["caller_ip"] = id.TailscaleIP
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

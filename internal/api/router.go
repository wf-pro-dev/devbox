package api

import (
	"encoding/json"
	"net/http"
	"os"
	"strconv"

	tailkit "github.com/wf-pro-dev/tailkit"

	"github.com/wf-pro-dev/devbox/internal/auth"
	"github.com/wf-pro-dev/devbox/internal/search"
	"github.com/wf-pro-dev/devbox/internal/storage"
	"github.com/wf-pro-dev/devbox/internal/version"
)

// NewRouter wires all handlers and returns the root http.Handler.
//
// The lc *local.Client parameter is gone. A single *tailkit.Server now
// provides everything: auth middleware (via tailkit.AuthMiddleware), peer
// discovery (via srv.Server.LocalClient()), and the tsnet dialler used
// by the send handler to reach tailkitd on remote nodes.
func NewRouter(srv *tailkit.Server, store *storage.Store, blobs *storage.BlobStore) http.Handler {
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
		srv:   srv,
	}
	sch := &searchHandler{
		searcher: searcher,
	}

	mux := http.NewServeMux()

	// ── Health ───────────────────────────────────────────────────────────────
	mux.HandleFunc("GET /health", handleHealth)

	// ── Files ────────────────────────────────────────────────────────────────
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

	// ── Dirs ─────────────────────────────────────────────────────────────────
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
	h = auth.Middleware(srv)(h)
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

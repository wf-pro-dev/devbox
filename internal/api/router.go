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
		filesHandler: *fh,
	}
	sh := &sendHandler{
		store: store,
		blobs: blobs,
		srv:   srv,
	}
	sch := &searchHandler{
		searcher: searcher,
	}

	drh := &driftHandler{
		versions: verSvc,
		store:    store,
		blobs:    blobs,
		srv:      srv,
	}

	root := http.NewServeMux()
	protected := http.NewServeMux()

	// ── Health ───────────────────────────────────────────────────────────────
	// Docker and local probes need a public readiness endpoint, so keep health
	// outside the tailnet auth middleware.
	root.HandleFunc("GET /health", handleHealth)

	// ── Files ────────────────────────────────────────────────────────────────
	protected.HandleFunc("GET /files", fh.handleList)
	protected.HandleFunc("POST /files", fh.handleUpload)
	protected.HandleFunc("GET /files/{id}", fh.handleGet)
	protected.HandleFunc("PUT /files/{id}", fh.handleUpdate)
	protected.HandleFunc("PATCH /files/{id}", fh.handleEditMeta)
	protected.HandleFunc("DELETE /files/{id}", fh.handleDelete)
	protected.HandleFunc("POST /files/{id}/tags", fh.handleAddTags)
	protected.HandleFunc("DELETE /files/{id}/tags/{tag}", fh.handleRemoveTag)
	protected.HandleFunc("POST /files/{id}/copy", fh.handleCopy)
	protected.HandleFunc("POST /files/{id}/move", fh.handleMove)
	protected.HandleFunc("GET /files/{id}/versions", fh.handleListVersions)
	protected.HandleFunc("GET /files/{id}/versions/{n}", fh.handleGetVersion)
	protected.HandleFunc("POST /files/{id}/versions/{n}/rollback", fh.handleRollback)
	protected.HandleFunc("GET /files/{id}/diff", fh.handleDiff)
	protected.HandleFunc("POST /files/{id}/send", sh.handleSendFile)
	protected.HandleFunc("GET /files/{id}/status", drh.handleGetFileStatus)
	protected.HandleFunc("GET /files/{id}/diff/node", drh.handleDiffNode)
	protected.HandleFunc("POST /files/{id}/diff/local", drh.handleDiffLocal)

	// ── Dirs ─────────────────────────────────────────────────────────────────
	protected.HandleFunc("GET /dirs", dh.handleList)
	protected.HandleFunc("POST /dirs", dh.handleCreate)
	protected.HandleFunc("GET /dirs/{dir}", dh.handleGet)
	protected.HandleFunc("PUT /dirs/{dir}", dh.handleSync)
	protected.HandleFunc("DELETE /dirs/{dir}", dh.handleDelete)
	protected.HandleFunc("POST /dirs/{dir}/tags", dh.handleAddTags)
	protected.HandleFunc("DELETE /dirs/{dir}/tags/{tag}", dh.handleRemoveTag)
	protected.HandleFunc("POST /dirs/{dir}/diff", dh.handleDiff)
	protected.HandleFunc("POST /dirs/{dir}/send", sh.handleSendDir)

	// ── Search ───────────────────────────────────────────────────────────────
	protected.HandleFunc("GET /search", sch.handleSearch)

	// ── Peers ────────────────────────────────────────────────────────────────
	protected.HandleFunc("GET /peers", sh.handleListPeers)

	root.Handle("/", auth.Middleware(srv)(protected))
	return root
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

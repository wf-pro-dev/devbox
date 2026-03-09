package api

import (
	"encoding/json"
	"net/http"
	"os"
	"strconv"

	"github.com/wf-pro-dev/devbox/internal/auth"
	"github.com/wf-pro-dev/devbox/internal/search"
	"github.com/wf-pro-dev/devbox/internal/storage"
	ver "github.com/wf-pro-dev/devbox/internal/version"
	"tailscale.com/client/local"
)

func NewRouter(lc *local.Client, store *storage.Store, blobs *storage.BlobStore) http.Handler {
	maxVersions := 10
	if v := os.Getenv("DEVBOX_MAX_VERSIONS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			maxVersions = n
		}
	}

	fh := &filesHandler{
		queries:  store.Queries,
		blobs:    blobs,
		searcher: search.New(store.DB),
	}

	dh := &dirHandler{
		queries:  store.Queries,
		blobs:    blobs,
		searcher: search.New(store.DB),
		lc:       lc,
	}

	dlh := &deliverHandler{
		queries: store.Queries,
		blobs:   blobs,
		lc:      lc,
	}

	uh := &versionHandler{
		queries: store.Queries,
		blobs:   blobs,
		version: ver.New(store.Queries, blobs, maxVersions),
	}

	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", handleHealth)

	// File routes
	mux.HandleFunc("GET /files", fh.handleListFiles)
	mux.HandleFunc("POST /files", fh.handleUploadFile)
	mux.HandleFunc("GET /files/{id}", fh.handleGetFile)
	mux.HandleFunc("DELETE /files/{id}", fh.handleDeleteFile)
	mux.HandleFunc("POST /files/{id}/tags", fh.handleAddTags)
	mux.HandleFunc("DELETE /files/{id}/tags/{tag}", fh.handleRemoveTag)
	mux.HandleFunc("POST /files/{id}/deliver", dlh.handleDeliver)
	mux.HandleFunc("PUT /files/{id}", fh.handleUpdateFile)
	mux.HandleFunc("PUT /files/{id}/versions", uh.handleVersionFile)
	mux.HandleFunc("GET /files/{id}/versions", uh.handleListVersions)

	// Directory routes
	mux.HandleFunc("GET /directories", dh.handleListDirs)
	mux.HandleFunc("POST /directories", dh.handleUploadDir)
	mux.HandleFunc("GET /directories/{id}", dh.handleGetDir)
	mux.HandleFunc("DELETE /directories/{id}", dh.handleDeleteDir)
	mux.HandleFunc("POST /directories/{id}/tags", dh.handleTagDir)
	mux.HandleFunc("POST /directories/{id}/deliver", dh.handleDeliverDir)
	mux.HandleFunc("PUT /directories/{id}", uh.handleUpdateDir)

	// Tailnet peers
	mux.HandleFunc("GET /peers", dlh.handleListPeers)

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

package api

import (
	"encoding/json"
	"net/http"

	"github.com/wf-pro-dev/devbox/internal/auth"
	"github.com/wf-pro-dev/devbox/internal/search"
	"github.com/wf-pro-dev/devbox/internal/storage"
	"tailscale.com/client/local"
)

// NewRouter sets up and returns the HTTP router.
func NewRouter(lc *local.Client, store *storage.Store, blobs *storage.BlobStore) http.Handler {
	fh := &filesHandler{
		queries:  store.Queries,
		blobs:    blobs,
		searcher: search.New(store.DB),
	}

	dh := &deliverHandler{
		queries: store.Queries,
		blobs:   blobs,
		lc:      lc,
	}

	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", handleHealth)

	// File routes
	mux.HandleFunc("GET /files", fh.handleListFiles)
	mux.HandleFunc("POST /files", fh.handleUploadFile)
	mux.HandleFunc("GET /files/{id}", fh.handleGetFile)
	mux.HandleFunc("DELETE /files/{id}", fh.handleDeleteFile)

	// Tag routes
	mux.HandleFunc("POST /files/{id}/tags", fh.handleAddTags)
	mux.HandleFunc("DELETE /files/{id}/tags/{tag}", fh.handleRemoveTag)

	// Delivery routes
	mux.HandleFunc("POST /files/{id}/deliver", dh.handleDeliver)
	mux.HandleFunc("GET /peers", dh.handleListPeers)

	var h http.Handler = mux
	h = auth.Middleware(lc)(h)

	return h
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	resp := map[string]string{
		"status":  "ok",
		"service": "devbox",
	}
	if id, ok := auth.FromContext(r.Context()); ok {
		resp["caller_host"] = id.Hostname
		resp["caller_user"] = id.UserLogin
		resp["caller_ip"] = id.TailscaleIP
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

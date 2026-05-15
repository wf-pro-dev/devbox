package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/wf-pro-dev/devbox/internal/models"
	"github.com/wf-pro-dev/tailkit"
)

type locationsHandler struct {
	srv *tailkit.Server
}

func (h *locationsHandler) handleGetLocations(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	locations, err := models.GetLocations(ctx, h.srv)
	if err != nil {
		jsonError(w, "failed to get locations", http.StatusInternalServerError)
		return
	}
	jsonOK(w, locations)
}

func (h *locationsHandler) handleGetLocationDirs(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	hostname := r.PathValue("hostname")
	prefix := r.URL.Query().Get("prefix")
	if prefix != "" {
		listing, err := models.GetLocationDir(ctx, h.srv, hostname, canonicalRemoteDir(prefix))
		if err != nil {
			jsonError(w, "failed to get location dir", http.StatusInternalServerError)
			return
		}
		jsonOK(w, listing)
		return
	}
	dirs, err := models.GetLocationDirs(ctx, h.srv, hostname)
	if err != nil {
		log.Printf("failed to get location dirs: %v", err)
		jsonError(w, fmt.Sprintf("failed to get location dirs: %v", err), http.StatusInternalServerError)
		return
	}
	jsonOK(w, dirs)
}

func (h *locationsHandler) handleGetLocationFile(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	hostname := r.PathValue("hostname")
	path := r.URL.Query().Get("path")
	if path == "" {
		var body struct {
			Path string `json:"path"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		path = body.Path
	}
	if path == "" {
		jsonError(w, "missing path", http.StatusBadRequest)
		return
	}

	file, content, err := models.ReadLocationFile(ctx, h.srv, hostname, canonicalRemoteFile(path))
	if err != nil {
		jsonError(w, "failed to get location file", http.StatusInternalServerError)
		return
	}

	if r.URL.Query().Get("meta") == "true" {
		jsonOK(w, file)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("X-File-Path", file.Path)
	w.Header().Set("X-File-Name", file.FileName)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(content))
}

func canonicalRemoteDir(path string) string {
	if path == "" || path == "/" {
		return "/"
	}
	clean := filepath.Clean(path)
	if !strings.HasPrefix(clean, "/") {
		clean = "/" + clean
	}
	if !strings.HasSuffix(clean, "/") {
		clean += "/"
	}
	return clean
}

func canonicalRemoteFile(path string) string {
	if path == "" {
		return "/"
	}
	clean := filepath.Clean(path)
	if !strings.HasPrefix(clean, "/") {
		clean = "/" + clean
	}
	return clean
}

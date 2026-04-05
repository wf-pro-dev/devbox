package api

import (
	"bytes"
	"io"
	"net/http"
	"slices"
	"strings"

	"github.com/wf-pro-dev/devbox/internal/storage"
	"github.com/wf-pro-dev/devbox/internal/version"
	"github.com/wf-pro-dev/tailkit"
	tailkitTypes "github.com/wf-pro-dev/tailkit/types"
)

// driftHandler handles API requests for file status and drift detection.
type driftHandler struct {
	versions *version.Service
	store    *storage.Store
	blobs    *storage.BlobStore
	fleet    *tailkit.Server
}

// GetFileStatus handles GET /files/:id/status.
// It compares a vault file against the tailkit fleet using SHA-256 hashes.
func (h *driftHandler) handleGetFileStatus(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	ctx := r.Context()

	peers, err := tailkit.OnlinePeers(ctx, h.fleet)
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	targetPeers := make([]tailkitTypes.Peer, 0, len(peers))

	nodes := r.URL.Query().Get("nodes")
	if nodes == "" {
		jsonError(w, "nodes parameter is required", http.StatusBadRequest)
		return
	}

	nodesSlice := strings.Split(nodes, ",")

	if len(nodesSlice) != 0 {
		for _, peer := range peers {
			if slices.Contains(nodesSlice, peer.Status.HostName) {
				targetPeers = append(targetPeers, peer)
			}
		}
	} else {
		targetPeers = peers
	}

	// 2. Delegate to the version service for drift logic
	results, err := h.versions.CheckFleetDrift(ctx, id, targetPeers)
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	jsonOK(w, results)
}

// DiffNode handles GET /files/:id/diff/node.
// It performs a line-by-line comparison between a vault version and a remote node.
func (h *driftHandler) handleDiffNode(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	node := r.URL.Query().Get("node")
	ver := r.URL.Query().Get("version") // Optional: defaults to "latest"

	if node == "" {
		jsonError(w, "node parameter is required", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	result, err := h.versions.DiffNodeContent(ctx, id, ver, node)
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	jsonOK(w, result)
}

// DiffLocal handles POST /files/:id/diff/local.
// It compares the current vault file against a file uploaded from the user's local machine.
func (h *driftHandler) handleDiffLocal(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	// 1. Get the uploaded file from the multipart request
	fileHeader, _, err := r.FormFile("file")
	if err != nil {
		jsonError(w, "local file upload required", http.StatusBadRequest)
		return
	}

	src, err := io.ReadAll(fileHeader)
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 2. Fetch the latest vault version for comparison
	fileMeta, err := h.store.ResolveFile(id)
	if err != nil {
		jsonError(w, err.Error(), http.StatusNotFound)
		return
	}

	vaultReader, err := h.blobs.Open(fileMeta.Sha256)
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer vaultReader.Close()

	// 3. Generate the Unified Diff
	result, err := h.versions.GenerateDiff(
		io.NopCloser(bytes.NewReader(src)),
		io.NopCloser(vaultReader),
		"vault:latest",
		"local:"+fileMeta.FileName,
	)
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	jsonOK(w, result)
}

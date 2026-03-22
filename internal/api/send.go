package api

import (
	"context"
	"encoding/json"
	"net/http"
	"path/filepath"
	"strings"

	tailkit "github.com/wf-pro-dev/tailkit"

	"github.com/wf-pro-dev/devbox/internal/db"
	"github.com/wf-pro-dev/devbox/internal/storage"
	"github.com/wf-pro-dev/devbox/internal/transfer"
	"github.com/wf-pro-dev/devbox/types"
)

// sendHandler handles file and directory delivery to tailnet peers via tailkitd.
//
// The lc *local.Client field is gone. Everything flows through *tailkit.Server:
//   - Peer discovery uses srv.Server.LocalClient().Status(ctx)
//   - The tsnet dialler for reaching tailkitd is srv.Server.Dial
type sendHandler struct {
	store *storage.Store
	blobs *storage.BlobStore
	srv   *tailkit.Server
}

// ── POST /files/{id}/send ─────────────────────────────────────────────────────

func (h *sendHandler) handleSendFile(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	file, err := h.store.ResolveFile(r.PathValue("id"))
	if err != nil {
		jsonError(w, err.Error(), http.StatusNotFound)
		return
	}

	var req types.SendRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid JSON body", http.StatusBadRequest)
		return
	}

	targets, err := h.resolveTargets(ctx, req)
	if err != nil || len(targets) == 0 {
		jsonError(w, "no valid targets", http.StatusBadRequest)
		return
	}

	results := transfer.Send(ctx, h.srv, transfer.SendPackage{
		FileID:     file.ID,
		FileName:   file.FileName,
		FilePath:   file.Path,
		BlobSha256: file.Sha256,
		BlobPath:   h.blobs.Path(file.Sha256),
		Targets:    targets,
		DestDir:    req.DestDir,
	})

	jsonOK(w, results)
}

// ── POST /dirs/{dir}/send ─────────────────────────────────────────────────────

func (h *sendHandler) handleSendDir(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	prefix := toPrefix(r.PathValue("dir"))

	files, err := h.store.Queries.ListFiles(ctx, db.ListFilesParams{Prefix: &prefix})
	if err != nil || len(files) == 0 {
		jsonError(w, "directory not found or empty", http.StatusNotFound)
		return
	}

	var req types.SendRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid JSON body", http.StatusBadRequest)
		return
	}

	targets, err := h.resolveTargets(ctx, req)
	if err != nil || len(targets) == 0 {
		jsonError(w, "no valid targets", http.StatusBadRequest)
		return
	}

	baseDir := req.DestDir
	if baseDir == "" {
		baseDir = transfer.DEFAULT_DEST_DIR + strings.TrimSuffix(prefix, "/")
	}

	type fileDelivery struct {
		Path    string               `json:"path"`
		Results []tailkit.SendResult `json:"results"`
	}

	var allResults []fileDelivery
	for _, f := range files {
		rel := strings.TrimPrefix(f.Path, prefix)
		destDir := baseDir
		if dir := filepath.Dir(rel); dir != "." {
			destDir = baseDir + "/" + dir
		}

		results := transfer.Send(ctx, h.srv, transfer.SendPackage{
			FileID:     f.ID,
			FileName:   f.FileName,
			BlobSha256: f.Sha256,
			BlobPath:   h.blobs.Path(f.Sha256),
			Targets:    targets,
			DestDir:    destDir,
		})

		allResults = append(allResults, fileDelivery{Path: f.Path, Results: results})
	}

	jsonOK(w, map[string]any{"prefix": prefix, "results": allResults})
}

// ── GET /peers ────────────────────────────────────────────────────────────────

func (h *sendHandler) handleListPeers(w http.ResponseWriter, r *http.Request) {
	// Peer discovery via the tailkit server's embedded tsnet LocalClient.
	lc, err := h.srv.Server.LocalClient()
	if err != nil {
		jsonError(w, "could not get local client", http.StatusInternalServerError)
		return
	}

	status, err := lc.Status(r.Context())
	if err != nil {
		jsonError(w, "could not list tailnet peers", http.StatusInternalServerError)
		return
	}

	type peer struct {
		Hostname string `json:"hostname"`
		DNSName  string `json:"dns_name"`
		IP       string `json:"ip"`
		Online   bool   `json:"online"`
	}

	var peers []peer
	for _, p := range status.Peer {
		dns := strings.TrimSuffix(p.DNSName, ".")
		short := strings.SplitN(dns, ".", 2)[0]
		ip := ""
		if len(p.TailscaleIPs) > 0 {
			ip = p.TailscaleIPs[0].String()
		}
		peers = append(peers, peer{Hostname: short, DNSName: dns, IP: ip, Online: p.Online})
	}
	if peers == nil {
		peers = []peer{}
	}
	jsonOK(w, peers)
}

// ── Helpers ───────────────────────────────────────────────────────────────────

// resolveTargets returns the list of target hostnames.
// When Broadcast is true all online tailnet peers are included.
// Peer discovery uses the tailkit server's LocalClient — no separate lc field.
func (h *sendHandler) resolveTargets(ctx context.Context, req types.SendRequest) ([]string, error) {
	if !req.Broadcast {
		return req.Targets, nil
	}

	lc, err := h.srv.Server.LocalClient()
	if err != nil {
		return nil, err
	}
	status, err := lc.Status(ctx)
	if err != nil {
		return nil, err
	}

	var targets []string
	for _, p := range status.Peer {
		if !p.Online {
			continue
		}
		dns := strings.TrimSuffix(p.DNSName, ".")
		if parts := strings.SplitN(dns, ".", 2); len(parts) > 0 {
			targets = append(targets, parts[0])
		}
	}
	return targets, nil
}

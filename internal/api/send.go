package api

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/wf-pro-dev/devbox/internal/db"
	"github.com/wf-pro-dev/devbox/internal/storage"
	"github.com/wf-pro-dev/devbox/internal/transfer"
	"github.com/wf-pro-dev/devbox/types"
	"tailscale.com/client/local"
)

type sendHandler struct {
	store *storage.Store
	blobs *storage.BlobStore
	lc    *local.Client
}

// ── POST /files/{id}/send ──────────────────────────────────────────────────────

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

	destDir := req.DestDir
	if destDir == "" {
		destDir = "~/devbox-received"
	}

	results := transfer.Send(ctx, h.lc, transfer.SendPackage{
		FileID:   file.ID,
		FileName: file.FileName,
		BlobPath: h.blobs.Path(file.Sha256),
		Targets:  targets,
		DestDir:  destDir,
	})

	resp := make([]types.SendResult, len(results))
	for i, res := range results {
		resp[i] = types.SendResult{Target: res.Target, Success: res.Err == nil}
		if res.Err != nil {
			resp[i].Error = res.Err.Error()
		}
	}
	jsonOK(w, map[string]any{"results": resp})
}

// ── POST /dirs/{dir}/deliver ──────────────────────────────────────────────────
// Delivers all files under the prefix to the target machines, preserving the
// relative subdirectory structure under dest_dir.

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
		baseDir = "~/devbox-received/" + strings.TrimSuffix(prefix, "/")
	}

	type fileDelivery struct {
		Path    string             `json:"path"`
		Results []types.SendResult `json:"results"`
	}

	var allResults []fileDelivery
	for _, f := range files {
		rel := strings.TrimPrefix(f.Path, prefix)
		destDir := baseDir
		if dir := filepath.Dir(rel); dir != "." {
			destDir = baseDir + "/" + dir
		}

		results := transfer.Send(ctx, h.lc, transfer.SendPackage{
			FileID:   f.ID,
			FileName: f.FileName,
			BlobPath: h.blobs.Path(f.Sha256),
			Targets:  targets,
			DestDir:  destDir,
		})

		dr := make([]types.SendResult, len(results))
		for i, res := range results {
			dr[i] = types.SendResult{Target: res.Target, Success: res.Err == nil}
			if res.Err != nil {
				dr[i].Error = res.Err.Error()
				log.Printf("deliver %s -> %s: %v", f.Path, res.Target, res.Err)
			}
		}
		allResults = append(allResults, fileDelivery{Path: f.Path, Results: dr})
	}

	jsonOK(w, map[string]any{
		"prefix":  prefix,
		"results": allResults,
	})
}

// ── GET /peers ────────────────────────────────────────────────────────────────

func (h *sendHandler) handleListPeers(w http.ResponseWriter, r *http.Request) {
	status, err := h.lc.Status(r.Context())
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

// ── helpers ───────────────────────────────────────────────────────────────────

func (h *sendHandler) resolveTargets(ctx context.Context, req types.SendRequest) ([]string, error) {
	if !req.Broadcast {
		return req.Targets, nil
	}
	status, err := h.lc.Status(ctx)
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

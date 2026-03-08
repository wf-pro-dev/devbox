package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/wf-pro-dev/devbox/internal/db"
	"github.com/wf-pro-dev/devbox/internal/transfer"
	"tailscale.com/client/local"
)

type deliverHandler struct {
	queries *db.Queries
	blobs   interface{ BlobPath(id string) string }
	lc      *local.Client
}

type deliverRequest struct {
	Targets   []string `json:"targets"`   // list of hostnames, or empty if Broadcast
	Broadcast bool     `json:"broadcast"` // send to all peers
	DestDir   string   `json:"dest_dir"`  // optional, default ~/devbox-received
}

type deliverResponse struct {
	Results []deliverResult `json:"results"`
}

type deliverResult struct {
	Target  string `json:"target"`
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

func (h *deliverHandler) handleDeliver(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := r.PathValue("id")

	file, err := h.queries.GetFile(ctx, id)
	if err != nil {
		if isNotFound(err) {
			jsonError(w, "file not found", http.StatusNotFound)
			return
		}
		jsonError(w, "failed to get file", http.StatusInternalServerError)
		return
	}

	var req deliverRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid JSON body", http.StatusBadRequest)
		return
	}

	targets, err := h.resolveTargets(ctx, req)
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}
	if len(targets) == 0 {
		jsonError(w, "no targets specified", http.StatusBadRequest)
		return
	}

	results := transfer.Deliver(ctx, h.lc, transfer.Delivery{
		FileID:   file.ID,
		FileName: file.Name,
		BlobPath: h.blobs.BlobPath(file.ID),
		Targets:  targets,
		DestDir:  req.DestDir,
	})

	resp := deliverResponse{Results: make([]deliverResult, len(results))}
	for i, res := range results {
		resp.Results[i] = deliverResult{
			Target:  res.Target,
			Success: res.Err == nil,
		}
		if res.Err != nil {
			resp.Results[i].Error = res.Err.Error()
		}
	}

	jsonOK(w, resp)
}

// handleListPeers handles GET /peers — returns all visible tailnet peers.
func (h *deliverHandler) handleListPeers(w http.ResponseWriter, r *http.Request) {
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

	peers := []peer{}
	for _, p := range status.Peer {
		dns := strings.TrimSuffix(p.DNSName, ".")
		short := dns
		if parts := strings.SplitN(dns, ".", 2); len(parts) > 0 {
			short = parts[0]
		}
		ip := ""
		if len(p.TailscaleIPs) > 0 {
			ip = p.TailscaleIPs[0].String()
		}
		peers = append(peers, peer{
			Hostname: short,
			DNSName:  dns,
			IP:       ip,
			Online:   p.Online,
		})
	}

	jsonOK(w, peers)
}

func (h *deliverHandler) resolveTargets(ctx context.Context, req deliverRequest) ([]string, error) {
	if !req.Broadcast {
		return req.Targets, nil
	}

	// Broadcast: resolve all online peers.
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

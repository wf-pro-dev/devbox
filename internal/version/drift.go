package version

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/pmezard/go-difflib/difflib"

	"github.com/wf-pro-dev/devbox/internal/db"
	"github.com/wf-pro-dev/tailkit"
	tailkitTypes "github.com/wf-pro-dev/tailkit/types"
)

// StatusResult represents the Layer 1 "Fast Check" for a single node.
type StatusResult struct {
	Hostname  string `json:"hostname"`
	Status    string `json:"status"` // MATCH (latest), MATCH (vN), DIFFERS, or NOT FOUND
	LocalPath string `json:"local_path"`
	Error     string `json:"error,omitempty"`
}

// DiffResult represents the Layer 2 "Deep Check" output.
type DiffResult struct {
	Unified    string `json:"unified"`
	Identical  bool   `json:"identical"`
	VaultLabel string `json:"vault_label"`
	NodeLabel  string `json:"node_label"`
}

// ── Layer 1: Fast Check (Status) ─────────────────────────────────────────────

// CheckFleetDrift performs concurrent comparisons across the fleet using tailkit fan-out.
func (s *Service) CheckFleetDrift(ctx context.Context, fileID string, peers []tailkitTypes.Peer) ([]StatusResult, error) {
	file, err := s.getFile(ctx, fileID)
	if err != nil {
		return nil, err
	}

	// 1. Fetch fleet configurations concurrently using the FleetClient
	fleet := tailkit.Nodes(nil, peers) // srv is nil as it's typically managed globally or passed
	configs, errs := fleet.Files().Config(ctx)

	var results []StatusResult
	for _, peer := range peers {
		node := peer.Status.HostName

		if err, ok := errs[node]; ok {
			results = append(results, StatusResult{Hostname: node, Error: err.Error()})
			continue
		}

		// 2. Verify path sharing using tailkit's MatchPathRule
		cfg := configs[node]
		rule, _, found := cfg.MatchPathRule(file.LocalPath)
		if !found || !rule.Share {
			continue // Path is not opted-in for auditing
		}

		// 3. Perform Layer 1 Stat check
		stat, err := tailkit.Node(nil, node).Files().Stat(ctx, file.LocalPath)
		if err != nil {
			results = append(results, StatusResult{Hostname: node, Status: "NOT FOUND", LocalPath: file.LocalPath})
			continue
		}

		status := "DIFFERS"
		if stat.SHA256 == file.Sha256 {
			status = "MATCH (latest)"
		} else if v, err := s.findMatchingVersion(ctx, file.ID, stat.SHA256); err == nil {
			status = fmt.Sprintf("MATCH (v%d)", v)
		}

		results = append(results, StatusResult{
			Hostname:  node,
			Status:    status,
			LocalPath: file.LocalPath,
		})
	}

	return results, nil
}

// ── Layer 2: Deep Check (Diff) ───────────────────────────────────────────────

// DiffNodeContent handles high-resolution diffs between vault and node.
func (s *Service) DiffNodeContent(ctx context.Context, fileID, versionStr, nodeName string) (DiffResult, error) {
	file, err := s.getFile(ctx, fileID)
	if err != nil {
		return DiffResult{}, err
	}

	// 1. Resolve Vault content from blob store
	targetSha, err := s.resolveShaForVersion(ctx, file, versionStr)
	if err != nil {
		return DiffResult{}, err
	}

	vaultReader, err := s.blobs.Open(targetSha)
	if err != nil {
		return DiffResult{}, err
	}
	defer vaultReader.Close()

	// 2. Fetch Remote Node content via tailkit
	// nodeReader is expected to be returned as an io.ReadCloser from a tailored tailkit helper
	nodeReader, err := s.openNodeReader(ctx, nodeName, file.LocalPath)
	if err != nil {
		return DiffResult{}, fmt.Errorf("node fetch: %w", err)
	}
	defer nodeReader.Close()

	// 3. Generate Unified Diff
	return s.GenerateDiff(vaultReader, nodeReader,
		fmt.Sprintf("devbox:%s@%s", file.ID, versionStr),
		fmt.Sprintf("node:%s:%s", nodeName, file.LocalPath),
	)
}

// ── Internal Helpers ─────────────────────────────────────────────────────────

func (s *Service) GenerateDiff(a, b io.Reader, labelA, labelB string) (DiffResult, error) {
	contentA, _ := io.ReadAll(a)
	contentB, _ := io.ReadAll(b)

	if string(contentA) == string(contentB) {
		return DiffResult{Identical: true, VaultLabel: labelA, NodeLabel: labelB}, nil
	}

	diff := difflib.UnifiedDiff{
		A:        difflib.SplitLines(string(contentA)),
		B:        difflib.SplitLines(string(contentB)),
		FromFile: labelA,
		ToFile:   labelB,
		Context:  3,
	}

	text, err := difflib.GetUnifiedDiffString(diff)
	return DiffResult{Unified: text, Identical: false, VaultLabel: labelA, NodeLabel: labelB}, err
}

func (s *Service) resolveShaForVersion(ctx context.Context, file db.File, vStr string) (string, error) {
	vStr = StripV(vStr) // Uses shared internal helper
	if vStr == "" || vStr == "latest" {
		return file.Sha256, nil
	}

	ver, err := s.queries.GetVersion(ctx, db.GetVersionParams{
		FileID:  file.ID,
		Version: parseVer(vStr),
	})
	if err != nil {
		return "", err
	}
	return ver.Sha256, nil
}

func (s *Service) findMatchingVersion(ctx context.Context, fileID, sha string) (int64, error) {
	versions, err := s.queries.ListVersions(ctx, db.ListVersionsParams{FileID: &fileID})
	if err != nil {
		return 0, err
	}
	for _, v := range versions {
		if v.Sha256 == sha {
			return v.Version, nil
		}
	}
	return 0, fmt.Errorf("no match")
}

func (s *Service) openNodeReader(ctx context.Context, node, path string) (io.ReadCloser, error) {
	// Helper to fetch content as a stream from the node
	content, err := tailkit.Node(nil, node).Files().Read(ctx, path)
	if err != nil {
		return nil, err
	}
	return io.NopCloser(strings.NewReader(content)), nil
}

func parseVer(s string) int64 {
	var i int64
	fmt.Sscanf(s, "%d", &i)
	return i
}

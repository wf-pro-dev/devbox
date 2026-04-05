// Package transfer delivers files to remote tailnet machines via tailkitd.
//
// Previously this package used SCP over SSH. It now sends files to tailkitd's
// POST /files endpoint, authenticated by Tailscale identity. The tsnet dialler
// from the *tailkit.Server is used so the connection goes through the
// WireGuard mesh and tailkitd sees a verified Tailscale identity via WhoIs.
//
// Peer hostname resolution is done via the tailkit server's LocalClient,
// removing the previous separate *local.Client parameter.
package transfer

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	tailkit "github.com/wf-pro-dev/tailkit"
	tailkitTypes "github.com/wf-pro-dev/tailkit/types"

	"github.com/wf-pro-dev/devbox/internal/storage"
)

const DEFAULT_DEST_DIR = "/var/lib/tailkitd/recv/"

// SendPackage describes a single file delivery to one or more machines.
type SendPackage struct {
	FileID     string
	FileName   string
	FilePath   string
	BlobSha256 string
	BlobPath   string   // absolute path to the zstd-compressed blob on disk
	Targets    []string // Tailscale short hostnames
	DestDir    string   // destination directory on the receiving node
}

// Result is the outcome of a single delivery attempt.
type Result struct {
	Target string
	Err    error
}

// Send delivers pkg to each target by calling POST /files on that node's
// tailkitd instance. The *tailkit.Server provides both:
//   - LocalClient for resolving peer hostnames to MagicDNS names
//   - The tsnet Dial function for making authenticated connections through
//     the tailnet without any extra ports or SSH keys
func Send(ctx context.Context, srv *tailkit.Server, pkg SendPackage) []tailkitTypes.SendResult {

	peers, err := resolvePeers(ctx, srv)
	if err != nil {
		log.Printf("transfer: peer list failed: %v — using targets as-is", err)
		peers = map[string]string{}
		for _, t := range pkg.Targets {
			peers[t] = t
		}
	}

	destPath := GetDestPath(pkg)

	body, err := storage.ReadBlob(pkg.BlobPath)
	if err != nil {
		return failResults(pkg.Targets, pkg.BlobPath, destPath, err)
	}

	TEMP_DIR := os.TempDir()
	tmp, err := os.CreateTemp(TEMP_DIR, ".tailkitd-recv-*")
	if err != nil {
		return failResults(pkg.Targets, pkg.BlobPath, destPath, err)
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath) // no-op after successful rename

	_, err = io.Copy(tmp, bytes.NewReader(body))
	if err != nil {
		_ = tmp.Close()
		return failResults(pkg.Targets, pkg.BlobPath, destPath, err)
	}

	results := make([]tailkitTypes.SendResult, 0, len(pkg.Targets))
	for _, target := range pkg.Targets {
		dnsName, ok := peers[strings.ToLower(target)]
		if !ok {
			dnsName = target
		}
		res, err := sendViaTailkitd(ctx, srv, dnsName, tmpPath, pkg)
		if err != nil {
			log.Printf("transfer: deliver to %s failed: %v", target, err)
		}
		results = append(results, *res)

	}
	return results
}

// sendViaTailkitd sends one file to tailkitd on the named node.
//
// The tailkitd tsnet hostname convention is "tailkitd-{node-short-name}", so
// for a devbox node named "laptop" the tailkitd address is
// "http://tailkitd-laptop/files".
//
// We use srv.Server.Dial (the tsnet dialler) so the TCP connection is
// established through the WireGuard tunnel. tailkitd calls lc.WhoIs on the
// inbound connection and sees us as a legitimate tailnet peer — no SSH key
// or separate credential is required.
func sendViaTailkitd(ctx context.Context, srv *tailkit.Server, dnsName, tmpPath string, pkg SendPackage) (*tailkitTypes.SendResult, error) {
	// Decompress the blob before sending — blobs are stored zstd-compressed
	// by the BlobStore but tailkitd expects raw file bytes.

	dest := GetDestPath(pkg)
	failResult := tailkitTypes.SendResult{
		ToolName:    "devbox",
		Filename:    pkg.FileName,
		LocalPath:   pkg.FilePath,
		WrittenTo:   dest,
		DestMachine: dnsName,
		Success:     false,
	}

	res, err := tailkit.Node(srv, dnsName).Files().Send(ctx, tailkitTypes.SendRequest{
		ToolName:  "devbox",
		LocalPath: tmpPath,
		DestPath:  dest,
		Filename:  pkg.FileName,
	})
	if err != nil {
		failResult.Error = err.Error()
		return &failResult, nil
	}

	return &res, nil
}

// resolvePeers returns a map of lowercase short hostname → full MagicDNS name
// for all online tailnet peers, using the tailkit server's LocalClient.
// Previously this accepted *local.Client directly; now it derives it from srv.
func resolvePeers(ctx context.Context, srv *tailkit.Server) (map[string]string, error) {
	lc, err := srv.Server.LocalClient()
	if err != nil {
		return nil, fmt.Errorf("local client: %w", err)
	}
	status, err := lc.Status(ctx)
	if err != nil {
		return nil, err
	}
	peers := make(map[string]string)
	for _, peer := range status.Peer {
		dns := strings.TrimSuffix(peer.DNSName, ".")
		if dns == "" {
			continue
		}
		short := strings.ToLower(strings.SplitN(dns, ".", 2)[0])
		os_hostname := strings.ToLower(peer.HostName)
		peers[short] = os_hostname
		peers[strings.ToLower(dns)] = os_hostname
		peers[os_hostname] = os_hostname
	}
	return peers, nil
}

// expandTilde replaces a leading "~/" with the user's home directory.
func expandTilde(path string) string {
	if !strings.HasPrefix(path, "~/") {
		return path
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return strings.Replace(path, "~", "$HOME", 1)
	}
	return home + path[1:]
}

func failResults(targets []string, localPath string, writtenTo string, err error) []tailkitTypes.SendResult {

	results := make([]tailkitTypes.SendResult, len(targets))
	for i := range results {
		results[i] = tailkitTypes.SendResult{
			LocalPath:   localPath,
			WrittenTo:   writtenTo,
			DestMachine: targets[i],
			Success:     false,
			Error:       err.Error(),
		}
	}
	return results
}

func GetDestPath(pkg SendPackage) string {
	if pkg.DestDir == "" {
		return ""
	}
	return expandTilde(pkg.DestDir) + "/" + filepath.Base(pkg.FileName)
}

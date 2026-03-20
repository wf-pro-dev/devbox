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

	"github.com/wf-pro-dev/devbox/internal/storage"
)

// SendPackage describes a single file delivery to one or more machines.
type SendPackage struct {
	FileID     string
	FileName   string
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
func Send(ctx context.Context, srv *tailkit.Server, pkg SendPackage) []tailkit.SendResult {
	if pkg.DestDir == "" {
		pkg.DestDir = "~/devbox-received"
	}

	peers, err := resolvePeers(ctx, srv)
	if err != nil {
		log.Printf("transfer: peer list failed: %v — using targets as-is", err)
		peers = map[string]string{}
		for _, t := range pkg.Targets {
			peers[t] = t
		}
	}

	results := make([]tailkit.SendResult, 0, len(pkg.Targets))
	for _, target := range pkg.Targets {
		dnsName, ok := peers[strings.ToLower(target)]
		if !ok {
			dnsName = target
		}
		res, err := sendViaTailkitd(ctx, srv, dnsName, pkg)
		results = append(results, *res)
		if err != nil {
			log.Printf("transfer: deliver to %s failed: %v", target, err)
		} else {
			log.Printf("transfer: delivered %s to %s:%s", pkg.FileName, target, pkg.DestDir)
		}
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
func sendViaTailkitd(ctx context.Context, srv *tailkit.Server, dnsName string, pkg SendPackage) (*tailkit.SendResult, error) {
	// Decompress the blob before sending — blobs are stored zstd-compressed
	// by the BlobStore but tailkitd expects raw file bytes.
	body, err := readBlob(pkg.BlobPath)
	if err != nil {
		return nil, fmt.Errorf("read blob: %w", err)
	}

	TEMP_DIR := os.TempDir()
	tmp, err := os.CreateTemp(TEMP_DIR, ".tailkitd-recv-*")
	if err != nil {
		return nil, fmt.Errorf("create temp file in %s: %w", TEMP_DIR, err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName) // no-op after successful rename

	_, err = io.Copy(tmp, bytes.NewReader(body))
	if err != nil {
		_ = tmp.Close()
		return nil, fmt.Errorf("write to temp file: %w", err)
	}

	destPath := expandTilde(pkg.DestDir) + "/" + filepath.Base(pkg.FileName)
	tailkitdHost := "tailkitd-" + shortName(dnsName)
	res, err := tailkit.Node(srv, tailkitdHost).Send(ctx, tailkit.SendRequest{
		LocalPath: fmt.Sprintf("%s/%s", TEMP_DIR, tmpName),
		DestPath:  destPath,
	})
	if err != nil {
		return nil, fmt.Errorf("send file: %w", err)
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
		peers[short] = dns
		peers[strings.ToLower(dns)] = dns
	}
	return peers, nil
}

// readBlob opens the zstd-compressed blob at blobPath, decompresses it, and
// returns the raw bytes. The BlobStore stores all blobs zstd-compressed but
// tailkitd expects raw file content.
func readBlob(blobPath string) ([]byte, error) {
	f, err := os.Open(blobPath)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", blobPath, err)
	}
	defer f.Close()

	rc, err := storage.DecompressFrom(f)
	if err != nil {
		return nil, fmt.Errorf("decompress %s: %w", blobPath, err)
	}
	defer rc.Close()

	return io.ReadAll(rc)
}

// shortName returns the first DNS label of a MagicDNS hostname, lowercased.
// "laptop.tail12345.ts.net." → "laptop"
func shortName(dnsName string) string {
	dnsName = strings.TrimSuffix(dnsName, ".")
	if idx := strings.Index(dnsName, "."); idx > 0 {
		return strings.ToLower(dnsName[:idx])
	}
	return strings.ToLower(dnsName)
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

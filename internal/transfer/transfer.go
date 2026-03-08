package transfer

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
	"tailscale.com/client/local"
)

// Delivery describes a file delivery to one or more machines.
type Delivery struct {
	FileID   string
	FileName string
	BlobPath string   // absolute path on the server filesystem
	Targets  []string // Tailscale hostnames or MagicDNS names
	DestDir  string   // destination directory on target (default: ~/devbox-received/)
}

// Result is the outcome of a single delivery attempt.
type Result struct {
	Target string
	Err    error
}

// Deliver pushes a file to each target machine via SCP over Tailscale SSH.
// Tailscale SSH must be enabled on each target machine.
// Returns a result per target — partial success is possible.
func Deliver(ctx context.Context, lc *local.Client, d Delivery) []Result {
	if d.DestDir == "" {
		d.DestDir = "~/devbox-received"
	}

	// Resolve hostnames to full MagicDNS names using the Tailscale peer list.
	peers, err := resolvePeers(ctx, lc)
	if err != nil {
		// If we can't list peers, try the names as-is.
		log.Printf("transfer: could not list peers: %v — using targets as-is", err)
		peers = map[string]string{}
		for _, t := range d.Targets {
			peers[t] = t
		}
	}

	results := make([]Result, 0, len(d.Targets))
	for _, target := range d.Targets {
		dnsName, ok := peers[strings.ToLower(target)]
		if !ok {
			// Not found in peer list — try it directly.
			dnsName = target
		}
		err := scpFile(ctx, dnsName, d.BlobPath, d.FileName, d.DestDir)
		results = append(results, Result{Target: target, Err: err})
		if err != nil {
			log.Printf("transfer: deliver to %s failed: %v", target, err)
		} else {
			log.Printf("transfer: delivered %s to %s:%s", d.FileName, target, d.DestDir)
		}
	}
	return results
}

// resolvePeers returns a map of lowercase hostname → full MagicDNS name
// for all peers visible on the tailnet.
func resolvePeers(ctx context.Context, lc *local.Client) (map[string]string, error) {
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
		// Short hostname = first label of MagicDNS name.
		short := strings.ToLower(strings.SplitN(dns, ".", 2)[0])
		peers[short] = dns
		peers[strings.ToLower(dns)] = dns
	}
	return peers, nil
}

// scpFile copies a local file to target:destDir/fileName via Tailscale SSH.
// Tailscale SSH authenticates automatically using the node's Tailscale identity.
func scpFile(ctx context.Context, target, blobPath, fileName, destDir string) error {
	// Tailscale SSH listens on port 22 of the MagicDNS name.
	addr := net.JoinHostPort(target, "22")

	config := &ssh.ClientConfig{
		// Tailscale SSH uses the OS username of the connecting process.
		User: currentUser(),
		Auth: []ssh.AuthMethod{
			// Tailscale SSH doesn't require credentials — the connection is
			// authenticated at the Tailscale layer. We still need to provide
			// an auth method to satisfy the SSH client; use an empty password.
			ssh.Password(""),
		},
		// Tailscale SSH certificates are verified by Tailscale — safe to skip
		// the host key check here since the connection is over the tailnet.
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), //nolint:gosec
		Timeout:         15 * time.Second,
	}

	dialCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	conn, err := dialContext(dialCtx, "tcp", addr, config)
	if err != nil {
		return fmt.Errorf("ssh dial %s: %w", target, err)
	}
	defer conn.Close()

	// Open the local blob file.
	f, err := os.Open(blobPath)
	if err != nil {
		return fmt.Errorf("open blob: %w", err)
	}
	defer f.Close()

	fi, err := f.Stat()
	if err != nil {
		return fmt.Errorf("stat blob: %w", err)
	}

	// Use SCP protocol over the SSH connection.
	return scpSend(conn, f, fi.Size(), fileName, destDir)
}

// scpSend implements the SCP sink protocol to copy a file over an SSH connection.
func scpSend(conn *ssh.Client, r io.Reader, size int64, fileName, destDir string) error {
	// Expand ~ on the remote side by running mkdir then scp.
	session, err := conn.NewSession()
	if err != nil {
		return fmt.Errorf("new session: %w", err)
	}
	defer session.Close()

	// Set up a pipe to send SCP data.
	w, err := session.StdinPipe()
	if err != nil {
		return fmt.Errorf("stdin pipe: %w", err)
	}

	// Start the remote SCP receiver.
	if err := session.Start(fmt.Sprintf("mkdir -p %s && scp -qt %s", destDir, destDir)); err != nil {
		return fmt.Errorf("start scp: %w", err)
	}

	// Send SCP file header.
	fmt.Fprintf(w, "C0644 %d %s\n", size, filepath.Base(fileName))

	// Send file content.
	if _, err := io.Copy(w, r); err != nil {
		return fmt.Errorf("copy: %w", err)
	}

	// SCP end-of-file marker.
	fmt.Fprint(w, "\x00")
	w.Close()

	return session.Wait()
}

// dialContext dials an SSH server with context cancellation support.
func dialContext(ctx context.Context, network, addr string, config *ssh.ClientConfig) (*ssh.Client, error) {
	d := net.Dialer{Timeout: config.Timeout}
	conn, err := d.DialContext(ctx, network, addr)
	if err != nil {
		return nil, err
	}
	c, chans, reqs, err := ssh.NewClientConn(conn, addr, config)
	if err != nil {
		return nil, err
	}
	return ssh.NewClient(c, chans, reqs), nil
}

func currentUser() string {
	if u := os.Getenv("USER"); u != "" {
		return u
	}
	return "root"
}

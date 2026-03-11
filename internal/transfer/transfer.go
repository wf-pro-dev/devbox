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

const (
	SSHKeyPath = "/run/secrets/devbox_ssh_key"
)

// Package describes a file delivery to one or more machines.
type SendPackage struct {
	FileID     string
	FileName   string
	BlobSha256 string
	BlobPath   string   // absolute path on the server filesystem
	Targets    []string // Tailscale hostnames or MagicDNS names
	DestDir    string   // destination directory on target (default: ~/devbox-received/)
}

// Result is the outcome of a single delivery attempt.
type Result struct {
	Target string
	Err    error
}

// Send pushes a file to each target machine via SCP over SSH.
func Send(ctx context.Context, lc *local.Client, s SendPackage) []Result {
	if s.DestDir == "" {
		s.DestDir = "~/devbox-received"
	}

	// Load the devbox private key once, reuse across all targets.
	signer, err := loadPrivateKey(SSHKeyPath)
	if err != nil {
		// Return the key load error for every target — nothing can proceed.
		results := make([]Result, len(s.Targets))
		for i, t := range s.Targets {
			results[i] = Result{Target: t, Err: fmt.Errorf("load ssh key: %w", err)}
		}
		return results
	}

	// Resolve hostnames to full MagicDNS names using the Tailscale peer list.
	peers, err := resolvePeers(ctx, lc)
	if err != nil {
		// If we can't list peers, try the names as-is.
		log.Printf("transfer: could not list peers: %v — using targets as-is", err)
		peers = map[string]string{}
		for _, t := range s.Targets {
			peers[t] = t
		}
	}

	results := make([]Result, 0, len(s.Targets))
	for _, target := range s.Targets {
		dnsName, ok := peers[strings.ToLower(target)]
		if !ok {
			// Not found in peer list — try it directly.
			dnsName = target
		}
		err := scpFile(ctx, dnsName, s.BlobPath, s.FileName, s.DestDir, signer)
		results = append(results, Result{Target: target, Err: err})
		if err != nil {
			log.Printf("transfer: deliver to %s failed: %v", target, err)
		} else {
			log.Printf("transfer: delivered %s to %s:%s", s.FileName, target, s.DestDir)
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

// scpFile copies a local file to target:destDir/fileName via SSH key auth.
func scpFile(ctx context.Context, target, blobPath, fileName, destDir string, signer ssh.Signer) error {

	addr := net.JoinHostPort(target, "22")

	config := &ssh.ClientConfig{
		User: currentUser(),
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
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
	session, err := conn.NewSession()
	if err != nil {
		return fmt.Errorf("new session: %w", err)
	}
	defer session.Close()

	// Set up pipes for the SCP handshake — we need both stdin and stdout.
	w, err := session.StdinPipe()
	if err != nil {
		return fmt.Errorf("stdin pipe: %w", err)
	}
	ack, err := session.StdoutPipe()
	if err != nil {
		return fmt.Errorf("stdout pipe: %w", err)
	}

	// Expand ~ explicitly so mkdir -p works regardless of the remote shell.
	expandedDir := strings.Replace(destDir, "~", "$HOME", 1)

	// Start the remote SCP receiver.
	if err := session.Start(fmt.Sprintf("mkdir -p %s && scp -qt %s", expandedDir, expandedDir)); err != nil {
		return fmt.Errorf("start scp: %w", err)
	}

	// Read the initial acknowledgment from the remote SCP process before sending.
	if err := readAck(ack); err != nil {
		return fmt.Errorf("initial ack: %w", err)
	}

	// Send SCP file header.
	fmt.Fprintf(w, "C0644 %d %s\n", size, filepath.Base(fileName))

	// Remote must acknowledge the header before we stream data.
	if err := readAck(ack); err != nil {
		return fmt.Errorf("header ack: %w", err)
	}

	// Send file content.
	if _, err := io.Copy(w, r); err != nil {
		return fmt.Errorf("copy: %w", err)
	}

	// SCP end-of-transfer marker, then wait for final ack.
	fmt.Fprint(w, "\x00")
	if err := readAck(ack); err != nil {
		return fmt.Errorf("final ack: %w", err)
	}

	w.Close()
	return session.Wait()
}

// readAck reads a single SCP acknowledgment byte from the remote.
// SCP uses 0x00 for success, 0x01 for warning, 0x02 for fatal error.
func readAck(r io.Reader) error {
	buf := make([]byte, 1)
	if _, err := io.ReadFull(r, buf); err != nil {
		return fmt.Errorf("read ack: %w", err)
	}
	switch buf[0] {
	case 0:
		return nil
	case 1, 2:
		// Read the error message that follows.
		msg, _ := io.ReadAll(r)
		return fmt.Errorf("scp remote error: %s", strings.TrimSpace(string(msg)))
	default:
		return fmt.Errorf("scp unexpected ack byte: %d", buf[0])
	}
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

// loadPrivateKey reads and parses a PEM-encoded private key from disk.
func loadPrivateKey(path string) (ssh.Signer, error) {
	keyBytes, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read key %s: %w", path, err)
	}
	signer, err := ssh.ParsePrivateKey(keyBytes)
	if err != nil {
		return nil, fmt.Errorf("parse key %s: %w", path, err)
	}
	return signer, nil
}

func currentUser() string {
	if u := os.Getenv("USER"); u != "" {
		return u
	}
	return "root"
}

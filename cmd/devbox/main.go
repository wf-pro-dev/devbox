package main

import (
	"log"
	"net"
	"net/http"
	"os"

	tailkit "github.com/wf-pro-dev/tailkit"

	"github.com/wf-pro-dev/devbox/internal/api"
	"github.com/wf-pro-dev/devbox/internal/storage"
)

func main() {
	log.Println("devbox starting...")

	hostname := os.Getenv("DEVBOX_HOSTNAME")
	if hostname == "" {
		hostname = "devbox"
	}

	// ── Database ────────────────────────────────────────────────────────────
	dbPath := os.Getenv("DEVBOX_DB_PATH")
	if dbPath == "" {
		dbPath = "./data/devbox.db"
	}
	store, err := storage.Open(dbPath)
	if err != nil {
		log.Fatalf("storage: %v", err)
	}
	defer store.DB.Close()

	// ── Blob store ──────────────────────────────────────────────────────────
	blobPath := os.Getenv("DEVBOX_STORAGE_PATH")
	if blobPath == "" {
		blobPath = "./data/blobs"
	}
	blobs, err := storage.NewBlobStore(blobPath, store.DB)
	if err != nil {
		log.Fatalf("blobstore: %v", err)
	}

	// ── Tailscale via tailkit ────────────────────────────────────────────────
	// tailkit.NewServer handles: tsnet.Server construction, auth key resolution
	// from env, TLS certificate provisioning via lc.GetCertificate, and
	// graceful shutdown on SIGTERM/SIGINT. No manual tsnet wiring needed.
	srv, err := tailkit.NewServer(tailkit.ServerConfig{
		Hostname: hostname,
		AuthKey:  os.Getenv("TS_AUTHKEY"),
		StateDir: "./data/tsnet-state",
	})
	if err != nil {
		log.Fatalf("tailkit server: %v", err)
	}
	defer srv.Close()

	// ── Router ──────────────────────────────────────────────────────────────
	// NewRouter receives only *tailkit.Server. Auth middleware, peer
	// discovery, and tsnet dialling for send all flow through it.
	router := api.NewRouter(srv, store, blobs)

	// ── Local dev listener ──────────────────────────────────────────────────
	// Plain HTTP on loopback for local development (Vite proxy, curl tests).
	// This is devbox-specific and not handled by tailkit.
	localAddr := os.Getenv("DEVBOX_LOCAL_ADDR")
	if localAddr == "" {
		localAddr = "127.0.0.1:8888"
	}
	localLn, err := net.Listen("tcp", localAddr)
	if err != nil {
		log.Fatalf("local listen: %v", err)
	}
	defer localLn.Close()

	go func() {
		log.Printf("devbox local listener on http://%s (localhost only)", localAddr)
		if err := http.Serve(localLn, router); err != nil {
			log.Fatalf("local server: %v", err)
		}
	}()

	// ── Tailnet HTTPS listener ───────────────────────────────────────────────
	// tailkit.Server.ListenAndServeTLS opens the tsnet TLS listener using
	// Tailscale-issued certificates and serves until the process exits.
	log.Printf("devbox listening on https://%s (tailnet)", hostname)
	if err := srv.ListenAndServeTLS(":443", router); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server: %v", err)
	}
}

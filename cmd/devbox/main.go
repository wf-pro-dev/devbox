package main

import (
	"log"
	"net"
	"net/http"
	"os"

	"github.com/wf-pro-dev/devbox/internal/api"
	"github.com/wf-pro-dev/devbox/internal/storage"
	"tailscale.com/tsnet"
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

	// ── Tailscale ───────────────────────────────────────────────────────────
	srv := &tsnet.Server{
		Hostname: hostname,
		AuthKey:  os.Getenv("TS_AUTHKEY"),
		Dir:      "./data/tsnet-state",
	}
	defer srv.Close()

	ln, err := srv.ListenTLS("tcp", ":443")
	if err != nil {
		log.Fatalf("tsnet listen error: %v", err)
	}
	defer ln.Close()

	httpsHost := hostname
	if domains := srv.CertDomains(); len(domains) > 0 {
		httpsHost = domains[0]
	}

	lc, err := srv.LocalClient()
	if err != nil {
		log.Fatalf("tsnet local client error: %v", err)
	}

	// ── Local dev listener ──────────────────────────────────────────────────
	localAddr := os.Getenv("DEVBOX_LOCAL_ADDR")
	if localAddr == "" {
		localAddr = "127.0.0.1:8888"
	}
	localLn, err := net.Listen("tcp", localAddr)
	if err != nil {
		log.Fatalf("local listen error: %v", err)
	}
	defer localLn.Close()

	// ── Router ──────────────────────────────────────────────────────────────
	router := api.NewRouter(lc, store, blobs)

	go func() {
		log.Printf("devbox local listener on http://%s (localhost only)", localAddr)
		if err := http.Serve(localLn, router); err != nil {
			log.Fatalf("local server error: %v", err)
		}
	}()

	log.Printf("devbox listening on https://%s (tailnet)", httpsHost)
	log.Printf("tailnet test : curl https://%s/health", httpsHost)
	log.Printf("local test   : curl http://%s/health", localAddr)

	if err := http.Serve(ln, router); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

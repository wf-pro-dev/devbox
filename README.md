# devbox

> A self-hosted developer toolbox for storing, tagging, and transferring files across machines — secured by your Tailscale network.

---

## What it is

**devbox** is a lightweight, self-hosted application that acts as a central hub for the files you reach for most as a developer: bash scripts, YAML configs, systemd units, dotfiles, SQL snippets, and anything else you want available on every machine you work from.

It runs as a single binary on any machine in your [Tailscale](https://tailscale.com) network. Any other machine on that tailnet can instantly browse, search, pull, or push files — no accounts, no passwords, no port-forwarding. Trust comes from the network.

---

## Features

- **File & snippet storage** — store bash scripts, YAML, TOML, configs, daemons, and plain text. Tag everything, search by content or tag.
- **Syntax-highlighted preview** — browse and review files from any machine in a browser.
- **Version history** — every file update is versioned. Roll back anytime.
- **Direct machine transfer** — send large files peer-to-peer between two tailnet machines via [croc](https://github.com/schollz/croc). Encrypted, resumable, LAN-aware.
- **Zero-config auth** — identity comes from Tailscale. No login form. Any machine on your tailnet is trusted.
- **CLI-first** — a `devbox` CLI lets you push, pull, search, and send without opening a browser.

---

## Tech stack

| Layer | Technology |
|---|---|
| Language | Go 1.22+ |
| HTTP framework | [Fiber v2](https://github.com/gofiber/fiber) |
| Network & auth | [Tailscale tsnet](https://pkg.go.dev/tailscale.com/tsnet) |
| File transfer | [croc](https://github.com/schollz/croc) (embedded) |
| Database | SQLite + FTS5 |
| Frontend | SvelteKit + CodeMirror 6 |
| Deployment | Single Docker container |

---

## Architecture overview

```
┌─────────────────────────────────────────────────┐
│              Tailscale Network                   │
│                                                  │
│   [machine-a]      [machine-b]      [machine-c]  │
│   browser/CLI  ←→  browser/CLI  ←→  devbox host  │
│                                                  │
│   ✦ WireGuard mesh, automatic TLS               │
│   ✦ Identity resolved from node keys            │
│   ✦ croc P2P transfer stays within tailnet      │
└─────────────────────────────────────────────────┘
            ↓ all traffic encrypted
┌─────────────────────────────────────────────────┐
│   devbox binary (tsnet + Fiber + SQLite)         │
│   ├── REST API                                   │
│   ├── Tailscale LocalAPI (identity resolution)   │
│   ├── croc agent (P2P transfer broker)           │
│   └── SvelteKit UI (served from embed.FS)        │
└─────────────────────────────────────────────────┘
```

---

## CLI usage (planned)

```bash
# Store a file with tags
devbox push deploy.sh --tag=bash,deploy

# List files by tag
devbox ls --tag=yaml

# Search by content
devbox search "postgres"

# Pull a file by ID
devbox pull abc123

# Pull and pipe directly
devbox pull abc123 | kubectl apply -f -

# Send a file directly to another machine (P2P)
devbox send --to=machine-b ./dump.sql
```

---

## Getting started

> 🚧 Work in progress — setup instructions will be added as the project is built.

Prerequisites:
- A machine with [Tailscale](https://tailscale.com/download) installed
- Docker (or Go 1.22+ to build from source)
- A Tailscale auth key (for `tsnet` startup)

```bash
# Clone
git clone https://github.com/wf-pro-dev/devbox.git
cd devbox

# Run (Docker)
docker run -e TS_AUTHKEY=<your-key> -v devbox-data:/data ghcr.io/wf-pro-dev/devbox

# Or build from source
go build ./cmd/devbox
TS_AUTHKEY=<your-key> ./devbox
```

Once running, open `https://devbox` from any machine on your tailnet.

---

## Project structure (planned)

```
devbox/
├── cmd/
│   ├── devbox/        # server entrypoint
│   └── devbox-cli/    # CLI client
├── internal/
│   ├── api/           # Fiber route handlers
│   ├── auth/          # Tailscale identity middleware
│   ├── storage/       # SQLite + blob storage layer
│   ├── transfer/      # croc integration
│   └── search/        # FTS5 query layer
├── web/               # SvelteKit frontend
├── Dockerfile
└── README.md
```

---

## License

MIT

# devbox

> A self-hosted developer toolbox for storing, tagging, and transferring files across machines — secured by your Tailscale network.

---

## What it is

**devbox** is a lightweight, self-hosted application that acts as a central hub for the files you reach for most as a developer: bash scripts, YAML configs, systemd units, dotfiles, SQL snippets, and anything else you want available on every machine you work from.

It runs as two containers (API backend + Nginx UI) on any machine in your [Tailscale](https://tailscale.com) network. Any other machine on that tailnet can instantly browse, search, pull, or push files — no accounts, no passwords, no port-forwarding. Trust comes from the network.

A `devbox-cli` binary gives you full access from the terminal on any machine.

---

## Features

- **File & snippet storage** — store bash scripts, YAML, TOML, configs, systemd units, and plain text. Tag everything, filter by language or tag, search by content.
- **Directory sync** — push, pull, diff, and sync entire local directories as named collections on the server.
- **Syntax-highlighted preview** — browse and review files from any machine in the browser with inline editing.
- **Version history** — every file update is versioned. View the diff, inspect old content, and roll back anytime.
- **Direct machine delivery** — send files or directories to one or all tailnet peers via [tailkitd](https://github.com/wf-pro-dev/tailkit).
- **Zero-config auth** — identity comes from Tailscale. No login form. Any machine on your tailnet is trusted.
- **CLI-first** — a `devbox-cli` covers every operation: push, pull, tag, diff, rollback, send, and more.

---

## Tech stack

| Layer | Technology |
|---|---|
| Language | Go 1.26+ |
| HTTP | Go stdlib `net/http` |
| Network & auth | [Tailscale tsnet](https://pkg.go.dev/tailscale.com/tsnet) via [tailkit](https://github.com/wf-pro-dev/tailkit) |
| File delivery | tailkitd (send over Tailscale mesh) |
| Database | SQLite + FTS5 |
| Blob storage | Content-addressable, zstd-compressed, on-disk |
| Frontend | SvelteKit + svelte-highlight |
| Deployment | Docker Compose (backend + Nginx) |

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
│   ✦ Identity resolved from Tailscale node keys  │
│   ✦ File delivery via tailkitd on each peer     │
└─────────────────────────────────────────────────┘
            ↓ all traffic encrypted
┌─────────────────────────────────────────────────┐
│   devbox-backend (tsnet + net/http + SQLite)     │
│   ├── REST API (/files, /dirs, /peers, /search)  │
│   ├── Tailscale LocalAPI (identity resolution)   │
│   └── tailkitd agent (file delivery broker)     │
├─────────────────────────────────────────────────┤
│   devbox-ui (SvelteKit, served via Nginx)        │
│   ├── File browser with syntax highlighting      │
│   ├── Directory tree view                        │
│   └── Send / deliver to peers                   │
└─────────────────────────────────────────────────┘
```

---

## Installation

### CLI (`devbox-cli`)

Install the CLI on any machine with one command:

```bash
curl -fsSL https://github.com/wf-pro-dev/devbox/releases/latest/download/install.sh | sh
```

Supports Linux and macOS on `amd64` and `arm64`. After install, register the node:

```bash
devbox-cli setup
```

Set your server URL so you don't have to pass it every time:

```bash
export DEVBOX_SERVER=https://devbox   # your devbox hostname on the tailnet
```

### Server

> **Note:** Public Docker images are coming with the first stable release. Until then, build from source — see [docs/deployment.md](docs/deployment.md).

Once images are published, the server installs with:

```bash
# 1. Copy the compose file
curl -fsSL https://github.com/wf-pro-dev/devbox/releases/latest/download/docker-compose.yml -o docker-compose.yml

# 2. Set your Tailscale auth key and start
TS_AUTHKEY=<your-key> docker compose up -d
```

See [docs/deployment.md](docs/deployment.md) for full setup, environment variables, and build-from-source instructions.

---

## CLI usage

```bash
# Upload a file with tags
devbox-cli files push deploy.sh --tags bash,deploy

# List files, filter by tag or language
devbox-cli files ls --tag deploy
devbox-cli files ls --lang yaml

# Download a file
devbox-cli files pull deploy.sh

# Push a whole directory
devbox-cli dirs push ./nginx --name nginx --tags infra

# Compare local dir against the server
devbox-cli dirs diff nginx ./nginx

# Sync changes back up
devbox-cli dirs update nginx ./nginx -m "update upstream block"

# Send a file to another machine on your tailnet
devbox-cli files send deploy.sh --to machine-b --dest /opt/scripts

# Send a directory to all online peers
devbox-cli dirs send nginx --all --dest /etc/nginx

# View version history and roll back
devbox-cli files log deploy.sh
devbox-cli files rollback deploy.sh 2

# List online tailnet peers
devbox-cli peers
```

For the full command reference see [docs/cli.md](docs/cli.md).

---

## Project structure

```
devbox/
├── cmd/
│   ├── devbox/            # server entrypoint
│   └── devbox-cli/        # CLI client
│       └── cmd/
│           ├── files/     # file subcommands
│           └── dirs/      # directory subcommands
├── internal/
│   ├── api/               # HTTP route handlers
│   ├── auth/              # Tailscale identity middleware
│   ├── models/            # shared business logic (files, tags)
│   ├── search/            # FTS5 query layer
│   ├── storage/           # SQLite store + CAS blob store
│   ├── transfer/          # tailkitd delivery
│   ├── version/           # versioning & rollback service
│   └── progress/          # upload progress tracking
├── db/queries/            # sqlc SQL queries
├── web/                   # SvelteKit frontend
├── docker/                # Dockerfiles + compose
├── install.sh             # one-line CLI installer
└── Makefile
```

---

## Docs

| Document | Description |
|---|---|
| [docs/cli.md](docs/cli.md) | Full CLI reference — all subcommands, flags, examples |
| [docs/deployment.md](docs/deployment.md) | Server setup, Docker Compose, environment variables |
| [docs/development.md](docs/development.md) | Build from source, local dev workflow, contributing |

---

## License

MIT

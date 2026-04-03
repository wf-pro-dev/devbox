# Deployment Guide

This guide covers everything required to run the devbox server and CLI. The server is made up of two containers:

- **`devbox-backend`** — the Go API server. Joins your Tailscale network via tsnet and manages data.
- **`devbox-ui`** — an Nginx container that serves the SvelteKit frontend and proxies API requests.

---

## 1. CLI Installation

Install the `devbox-cli` on any machine you want to interact with the server from.

```bash
curl -fsSL https://github.com/wf-pro-dev/devbox/releases/latest/download/install.sh | sh
````

After installing, set your server URL so you don't have to pass it manually on every command:

```bash
export DEVBOX_SERVER=https://devbox   # Replace 'devbox' with your Tailscale hostname
```

*(Optional)* Run `devbox-cli setup` to register the node for tailkitd tool discovery.

-----

## 2\. Server Deployment Options

Devbox offers two deployment paths depending on your architecture needs.

### Option A: Primary Setup (Unified)

This runs both the backend and UI on the same machine using a single `docker-compose.yml` file.

**Why choose this setup?**

  - Easiest "one-command" installation.
  - Improved security: The backend API port is not exposed to the host machine; the UI proxies traffic internally over Docker's bridge network.
  - Zero-config networking between the frontend and backend.

**Requirements:**

  - A machine with Docker and Docker Compose installed.
  - A [Tailscale Auth Key](https://login.tailscale.com/admin/settings/keys) (a reusable auth key is recommended).

**Instructions:**

```bash
# 1. Download the unified compose file
curl -fsSL https://github.com/wf-pro-dev/devbox/releases/latest/download/docker-compose.yml -o docker-compose.yml

# 2. Start the services
export TS_AUTHKEY="tskey-auth-..."
docker compose up -d
```

### Option B: Advanced Setup (Decoupled)

Because the UI is stateless, you can run the backend and UI on entirely separate machines.

**Why choose this setup?**

  - You want to securely host the heavy `backend` (and its SQLite/blob data) on a secure home NAS or private server.
  - You want to host the lightweight `ui` somewhere public-facing, on a cheap cloud VPS, or at the edge.

**Requirements:**

  - Docker and Docker Compose on the backend host.
  - Docker on the UI host.
  - A [Tailscale Auth Key](https://login.tailscale.com/admin/settings/keys).
  - Network access between the UI container and the Backend's exposed API port.

**Instructions:**

1.  **Start the Backend:**
    Download the backend-only compose file and start it. This exposes port `8888` explicitly.
    ```bash
    curl -fsSL https://github.com/wf-pro-dev/devbox/releases/latest/download/docker-compose.backend.yml -o docker-compose.yml
    export TS_AUTHKEY="tskey-auth-..."
    docker compose up -d
    ```
2.  **Start the UI (on any machine):**
    Run the UI container, passing the IP/Domain of your backend instance via the `BACKEND_URL` environment variable.
    ```bash
    docker run -d -p 80:80 \
      -e BACKEND_URL="http://<YOUR_BACKEND_IP>:8888" \
      ghcr.io/wf-pro-dev/devbox/ui:latest
    ```

-----

## Environment Variables

### Backend

| Variable | Default | Description |
|---|---|---|
| `TS_AUTHKEY` | *(required)* | Tailscale auth key used by tsnet to join the tailnet |
| `DEVBOX_HOSTNAME` | `devbox` | The hostname devbox registers on the tailnet. Accessible at `https://<hostname>` from any peer. |
| `DEVBOX_DB_PATH` | `/data/devbox.db` | Path to the SQLite database file |
| `DEVBOX_STORAGE_PATH` | `/data/blobs` | Path to the blob storage directory (content-addressable, zstd-compressed) |
| `DEVBOX_LOCAL_ADDR` | `127.0.0.1:8888` | Local HTTP listener address. Needs to be `0.0.0.0:8888` if accepting external UI connections. |
| `DEVBOX_MAX_VERSIONS` | `10` | Maximum number of versions to keep per file. Older versions are pruned automatically. |

### UI (Nginx)

| Variable | Default | Description |
|---|---|---|
| `BACKEND_URL` | *(none)* | Only required if running decoupled. Instructs Nginx where to proxy API requests (e.g., `http://192.168.1.50:8888`). |

-----

## Data Persistence

All server state lives in the `devbox-data` Docker volume, mounted at `/data` inside the backend container:

```text
/data/
├── devbox.db          # SQLite database (files, versions, tags, blobs table)
├── blobs/             # Content-addressable blob store
│   └── <sha[0:2]>/
│       └── <sha[2:4]>/
│           └── <full-sha>   # zstd-compressed file content
└── tsnet-state/       # Tailscale node state (certificates, keys)
```

Back up the entire volume to preserve all files, history, and the Tailscale node identity.

-----

## Accessing the UI

Once running, open a browser on any machine on your tailnet and navigate to:

```text
https://devbox        # or whatever you set DEVBOX_HOSTNAME to
```

Tailscale issues TLS certificates automatically — no self-signed cert warnings.

-----

## Health Check

The backend exposes a health endpoint.

From the Docker host:

```bash
curl http://localhost:8888/health
# {"status":"ok","service":"devbox"}
```

From within the tailnet:

```bash
curl https://devbox/health
# {"status":"ok","service":"devbox","caller_host":"your-machine","caller_ip":"100.x.x.x"}
```

-----

## Upgrading

Because your deployment relies on standard GHCR images, upgrading is simple:

```bash
# Pull the latest images
docker compose pull

# Restart with zero-downtime (data volume is preserved)
docker compose up -d
```

The SQLite schema uses `CREATE TABLE IF NOT EXISTS` and `CREATE INDEX IF NOT EXISTS` throughout, so new schema additions apply automatically on startup without a manual migration step.

-----
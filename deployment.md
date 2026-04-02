# Deployment

This guide covers running the devbox server. The server is made up of two containers:

- **`devbox-backend`** — the Go API server. Joins your Tailscale network via tsnet and listens on port `443` (tailnet TLS) and `8888` (local HTTP for the UI proxy).
- **`devbox-ui`** — a Nginx container that serves the SvelteKit frontend and proxies API requests to the backend.

---

## Prerequisites

- A machine with [Tailscale](https://tailscale.com/download) installed and running
- Docker and Docker Compose
- A Tailscale auth key — generate one at [tailscale.com/admin/settings/keys](https://login.tailscale.com/admin/settings/keys). A reusable auth key works for persistent deployments.
- **tailkitd** running on any machine you want to deliver files *to* (not required on the devbox host itself). See [tailkit](https://github.com/wf-pro-dev/tailkit) for installation.

---

## Quick start

> **Public images are not yet published.** Until the first public release, follow the [Build from source](#build-from-source) section to build your own images, then return here.
>
> Once images are available the quick start will be:
>
> ```bash
> curl -fsSL https://github.com/wf-pro-dev/devbox/releases/latest/download/docker-compose.yml -o docker-compose.yml
> TS_AUTHKEY=<your-key> DEVBOX_HOSTNAME=devbox docker compose up -d
> ```

---

## Build from source

```bash
# Clone the repo
git clone https://github.com/wf-pro-dev/devbox.git
cd devbox

# Build both images
make docker-build

# Start the server
TS_AUTHKEY=<your-key> docker compose -f docker/docker-compose.yml up -d
```

The compose file expects the images to exist locally. `make docker-build` builds and tags them.

---

## Docker Compose reference

The compose file lives at `docker/docker-compose.yml`. A minimal production deployment:

```yaml
services:
  backend:
    image: devbox/backend:latest   # or your registry path
    restart: unless-stopped
    environment:
      TS_AUTHKEY: ${TS_AUTHKEY}
      DEVBOX_HOSTNAME: ${DEVBOX_HOSTNAME:-devbox}
    volumes:
      - devbox-data:/data
    devices:
      - /dev/net/tun:/dev/net/tun
    cap_add:
      - NET_ADMIN
      - NET_RAW

  ui:
    image: devbox/ui:latest        # or your registry path
    restart: unless-stopped
    ports:
      - "80:80"
    depends_on:
      - backend

volumes:
  devbox-data:
```

The backend requires `NET_ADMIN` and `NET_RAW` capabilities plus access to `/dev/net/tun` so that tsnet can create a WireGuard interface inside the container.

---

## Environment variables

### Backend

| Variable | Default | Description |
|---|---|---|
| `TS_AUTHKEY` | *(required)* | Tailscale auth key used by tsnet to join the tailnet |
| `DEVBOX_HOSTNAME` | `devbox` | The hostname devbox registers on the tailnet. Accessible at `https://<hostname>` from any peer. |
| `DEVBOX_DB_PATH` | `/data/devbox.db` | Path to the SQLite database file |
| `DEVBOX_STORAGE_PATH` | `/data/blobs` | Path to the blob storage directory (content-addressable, zstd-compressed) |
| `DEVBOX_LOCAL_ADDR` | `127.0.0.1:8888` | Local HTTP listener address. Set to `0.0.0.0:8888` in Docker so Nginx can reach it. |
| `DEVBOX_MAX_VERSIONS` | `10` | Maximum number of versions to keep per file. Older versions are pruned automatically. |

### UI (Nginx)

The UI container has no application-level environment variables. Nginx is pre-configured to proxy `/files`, `/dirs`, `/peers`, `/health`, and `/search` to `http://backend:8888`.

---

## Data persistence

All server state lives in the `devbox-data` Docker volume, mounted at `/data`:

```
/data/
├── devbox.db          # SQLite database (files, versions, tags, blobs table)
├── blobs/             # Content-addressable blob store
│   └── <sha[0:2]>/
│       └── <sha[2:4]>/
│           └── <full-sha>   # zstd-compressed file content
└── tsnet-state/       # Tailscale node state (certificates, keys)
```

Back up the entire volume to preserve all files, history, and the Tailscale node identity.

---

## Accessing the UI

Once running, open a browser on any machine on your tailnet and navigate to:

```
https://devbox        # or whatever you set DEVBOX_HOSTNAME to
```

Tailscale issues TLS certificates automatically — no self-signed cert warnings.

The UI is also accessible on port `80` of the host machine running Docker, via the Nginx container, for local access without Tailscale.

---

## Health check

The backend exposes a health endpoint:

```bash
curl http://localhost:8888/health
# {"status":"ok","service":"devbox"}
```

From within the tailnet:

```bash
curl https://devbox/health
# {"status":"ok","service":"devbox","caller_host":"your-machine","caller_ip":"100.x.x.x"}
```

---

## Upgrading

```bash
# Pull the latest images (once public images are available)
docker compose -f docker/docker-compose.yml pull

# Or rebuild from source
git pull && make docker-build

# Restart with zero-downtime (data volume is preserved)
docker compose -f docker/docker-compose.yml up -d
```

The SQLite schema uses `CREATE TABLE IF NOT EXISTS` and `CREATE INDEX IF NOT EXISTS` throughout, so new schema additions apply automatically on startup without a migration step.

---

# CLI Reference

`devbox-cli` is the command-line interface for your self-hosted devbox server. Files and directories are addressed by UUID (or short prefix), full path, or unique filename. When a name is ambiguous the CLI will tell you and ask for the full path or ID.

---

## Global flags

| Flag | Default | Description |
|---|---|---|
| `--server <url>` | `$DEVBOX_SERVER` | devbox server URL. Overrides the environment variable. |

Set `DEVBOX_SERVER` once so you never have to pass `--server`:

```bash
export DEVBOX_SERVER=https://devbox   # your devbox hostname on the tailnet
```

---

## Setup

### `devbox-cli setup`

Register this machine with `tailkitd`. Run once after installing the CLI — required for the `send` commands to work. ! IMPORTATNT. You can skip this step. This is for [tailkit](https://github.com/wf-pro-dev/tailkit) / [tailkitd](https://github.com/wf-pro-dev/tailkit) tool discovery (NOT IMPLEMENTED YET)

```bash
devbox-cli setup
```

---

## Files

All file operations live under `devbox-cli files`.

### `files ls`

List files stored on the server. Combine filters freely.

```bash
devbox-cli files ls
devbox-cli files ls --tag deploy
devbox-cli files ls --lang bash
devbox-cli files ls --dir nginx
```

| Flag | Description |
|---|---|
| `--tag <name>` | Filter by tag |
| `--lang <name>` | Filter by language (e.g. `bash`, `yaml`, `go`) |
| `--dir <name>` | Filter by directory name or ID |

---

### `files push`

Upload a file to the server.

```bash
devbox-cli files push deploy.sh
devbox-cli files push deploy.sh --tags bash,deploy --lang bash
devbox-cli files push nginx/default.conf --dir nginx --path nginx/conf.d/default.conf
```

| Flag | Description |
|---|---|
| `--desc <text>` | Description |
| `--lang <name>` | Language (auto-detected from extension if omitted) |
| `--tags <list>` | Comma-separated tags |
| `--dir <name>` | Target directory name or ID |
| `--path <path>` | Logical path on server (defaults to filename) |

---

### `files pull`

Download a file to disk.

```bash
devbox-cli files pull deploy.sh
devbox-cli files pull abcd1234
devbox-cli files pull nginx/conf.d/default.conf --out /tmp/
devbox-cli files pull deploy.sh --version 2
```

| Flag | Description |
|---|---|
| `--out <path>` | Output path or directory (defaults to filename) |
| `--version <n>` | Download a specific version number |

---

### `files info`

Show full metadata for a file.

```bash
devbox-cli files info deploy.sh
devbox-cli files info abcd1234
```

Displays: ID, name, path, version, size, SHA-256, language, tags, description, uploaded by, created, updated.

---

### `files update`

Replace a file's content. Creates a new version if content has changed; does nothing if content is identical.

```bash
devbox-cli files update deploy.sh ./deploy.sh
devbox-cli files update abcd1234 ./new-deploy.sh -m "fix: correct db host"
```

| Flag | Description |
|---|---|
| `-m, --message <text>` | Version message (optional) |

---

### `files edit`

Edit file metadata without touching content.

```bash
devbox-cli files edit deploy.sh --desc "Production deploy script"
devbox-cli files edit abcd1234 --lang bash
devbox-cli files edit old/path.sh --path new/path.sh
```

| Flag | Description |
|---|---|
| `--desc <text>` | New description |
| `--lang <name>` | New language |
| `--path <path>` | New path (rename / move) |

At least one flag is required.

---

### `files mv`

Rename or move a file to a new path.

```bash
devbox-cli files mv deploy.sh scripts/deploy.sh
devbox-cli files mv abcd1234 nginx/deploy.sh
```

---

### `files cp`

Copy a file to a new path. The underlying blob is shared — no disk copy is made.

```bash
devbox-cli files cp deploy.sh deploy-backup.sh
devbox-cli files cp abcd1234 nginx/deploy.sh
```

---

### `files delete` / `files rm`

Delete a file. Prompts for confirmation unless `--force` is passed.

```bash
devbox-cli files delete deploy.sh
devbox-cli files delete abcd1234 --force
```

| Flag | Description |
|---|---|
| `-f, --force` | Skip confirmation prompt |

---

### `files tag`

Add a tag to a file.

```bash
devbox-cli files tag deploy.sh prod
devbox-cli files tag abcd1234 nginx
```

---

### `files untag`

Remove a tag from a file.

```bash
devbox-cli files untag deploy.sh prod
devbox-cli files untag abcd1234 nginx
```

---

### `files log`

Show version history for a file.

```bash
devbox-cli files log deploy.sh
devbox-cli files log abcd1234
```

Displays each version: number, size, SHA-256 (short), date, and message.

---

### `files diff`

Compare versions or a local file against the stored version.

```bash
# Current version vs previous version
devbox-cli files diff deploy.sh

# Specific version comparison
devbox-cli files diff deploy.sh v2 v1

# Local file vs stored version
devbox-cli files diff deploy.sh ./deploy.sh
```

---

### `files rollback`

Restore a file to a previous version. Prompts for confirmation unless `--force` is passed.

```bash
devbox-cli files rollback deploy.sh 2
devbox-cli files rollback deploy.sh v2 --force
```

| Flag | Description |
|---|---|
| `-f, --force` | Skip confirmation prompt |

---

### `files send`

Deliver a file to one or more tailnet peers via tailkitd.

```bash
devbox-cli files send deploy.sh --to machine-b
devbox-cli files send deploy.sh --to host1,host2 --dest /opt/scripts
devbox-cli files send deploy.sh --all
```

| Flag | Description |
|---|---|
| `--to <hosts>` | Comma-separated target hostnames |
| `--dest <dir>` | Destination directory on the target machine |
| `--all` | Deliver to all currently online peers |

Either `--to` or `--all` is required.

---

## Directories

Directories are virtual groups of files that share a path prefix (e.g. `nginx/` owns all files whose path starts with `nginx/`). Directory names are unique.

All directory operations live under `devbox-cli dirs`.

### `dirs ls`

List all directories on the server.

```bash
devbox-cli dirs ls
devbox-cli dirs ls --tag infra
```

| Flag | Description |
|---|---|
| `--tag <name>` | Filter to directories that contain at least one file with this tag |

---

### `dirs push`

Upload a local directory as a new named collection on the server.

```bash
devbox-cli dirs push ./nginx --name nginx
devbox-cli dirs push ./scripts --name scripts --desc "Deploy scripts" --tags prod,deploy
```

| Flag | Description |
|---|---|
| `--name <name>` | Directory name on the server (defaults to local folder name) |
| `--desc <text>` | Description |
| `--tags <list>` | Comma-separated tags applied to every uploaded file |

---

### `dirs pull`

Download a directory, recreating the directory structure locally.

```bash
devbox-cli dirs pull nginx
devbox-cli dirs pull nginx --out /tmp/nginx-backup
```

| Flag | Description |
|---|---|
| `--out <dir>` | Output directory (defaults to directory name) |

---

### `dirs update`

Update a local directory to an existing collection on the server. New files are added; existing files with changed content get a new version; files only on the server are left untouched.

```bash
devbox-cli dirs update nginx ./nginx
devbox-cli dirs update nginx ./nginx -m "update upstream block"
```

| Flag | Description |
|---|---|
| `-m, --message <text>` | Version message applied to updated files |

---

### `dirs diff`

Compare a local directory against the server collection. Shows added, changed, and server-only files.

```bash
devbox-cli dirs diff nginx ./nginx
devbox-cli dirs diff abcd1234 ./nginx
```

---

### `dirs delete` / `dirs rm`

Delete a directory and all its files. Prompts for confirmation unless `--force` is passed.

```bash
devbox-cli dirs delete nginx
devbox-cli dirs delete nginx --force
```

| Flag | Description |
|---|---|
| `-f, --force` | Skip confirmation prompt |

---

### `dirs tag`

Add a tag to all files in a directory.

```bash
devbox-cli dirs tag nginx prod
devbox-cli dirs tag abcd1234 infra
```

---

### `dirs untag`

Remove a tag from all files in a directory.

```bash
devbox-cli dirs untag nginx prod
```

---

### `dirs send`

Deliver an entire directory to one or more tailnet peers via tailkitd. The directory structure is preserved on the target.

```bash
devbox-cli dirs send nginx --to myhost
devbox-cli dirs send nginx --to host1,host2 --dest /etc/nginx
devbox-cli dirs send nginx --all
```

| Flag | Description |
|---|---|
| `--to <hosts>` | Comma-separated target hostnames |
| `--dest <dir>` | Destination directory on the target machine |
| `--all` | Deliver to all currently online peers |

Either `--to` or `--all` is required.

---

## Addressing files

Every command that accepts `<id|path>` resolves in this order:

1. Exact UUID match
2. Short UUID prefix (first 8 characters)
3. Exact full path match (e.g. `nginx/conf.d/default.conf`)
4. Exact filename match — returns an error if more than one file shares that name

When a filename is ambiguous the CLI prints all matching paths and asks you to use the full path or ID instead.

-- devbox schema v2
-- Storage model: content-addressable blobs (sha256-keyed, zstd-compressed).
-- Files are addressed by their full path. Collections are named bookmarks
-- that own a path prefix (e.g. collection "nginx" owns prefix "nginx/").


-- ── files ─────────────────────────────────────────────────────────────────────
-- path       : full logical path, e.g. "nginx/conf.d/default.conf" — globally unique
-- file_name  : basename only,    e.g. "default.conf"
-- sha256: CAS key — points to blobs/{sha[0:2]}/{sha[2:4]}/{sha} on disk
-- version    : current version number (bumped on every content change)
CREATE TABLE IF NOT EXISTS files (
    id            TEXT    PRIMARY KEY,
    path          TEXT    NOT NULL UNIQUE,
    local_path    TEXT    NOT NULL DEFAULT '',
    file_name     TEXT    NOT NULL DEFAULT '',
    description   TEXT    NOT NULL DEFAULT '',
    language      TEXT    NOT NULL DEFAULT '',
    size          INTEGER NOT NULL DEFAULT 0,
    sha256   TEXT    NOT NULL,
    uploaded_by   TEXT    NOT NULL DEFAULT '',
    version       INTEGER NOT NULL DEFAULT 1,
    created_at    TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now')),
    updated_at    TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now'))
);

-- ── blobs ─────────────────────────────────────────────────────────────────────
-- Reference-counted content store. One row per unique sha256.
-- ref_count: number of files+versions currently pointing at this blob.
-- When ref_count drops to 0 the blob file on disk can be deleted.
CREATE TABLE IF NOT EXISTS blobs (
    sha256      TEXT    PRIMARY KEY,
    size        INTEGER NOT NULL DEFAULT 0,
    ref_count   INTEGER NOT NULL DEFAULT 1,
    created_at  TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now'))
);

-- ── tags ──────────────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS tags (
    id   INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT    NOT NULL UNIQUE
);

-- ── file_tags ─────────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS file_tags (
    file_id TEXT    NOT NULL REFERENCES files(id)  ON DELETE CASCADE,
    tag_id  INTEGER NOT NULL REFERENCES tags(id)   ON DELETE CASCADE,
    PRIMARY KEY (file_id, tag_id)
);


-- ── versions ──────────────────────────────────────────────────────────────────
-- Each row is a snapshot of a file at a specific version.
-- sha256 references the blobs table (CAS key) — no file copy needed.
-- Rollback = point files.sha256 back at an existing blob row.
CREATE TABLE IF NOT EXISTS versions (
    id             INTEGER PRIMARY KEY AUTOINCREMENT,
    file_id        TEXT    NOT NULL REFERENCES files(id) ON DELETE CASCADE,
    version        INTEGER NOT NULL,
    sha256         TEXT    NOT NULL REFERENCES blobs(sha256),
    size           INTEGER NOT NULL DEFAULT 0,
    uploaded_by    TEXT    NOT NULL DEFAULT '',
    message        TEXT    NOT NULL DEFAULT '',
    created_at     TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now')),
    UNIQUE (file_id, version)
);

-- ── transfers ─────────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS transfers (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    from_host   TEXT    NOT NULL,
    to_host     TEXT    NOT NULL,
    file_path   TEXT    NOT NULL,
    size        INTEGER NOT NULL DEFAULT 0,
    duration_ms INTEGER NOT NULL DEFAULT 0,
    created_at  TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now'))
);

-- ── indexes ───────────────────────────────────────────────────────────────────
CREATE INDEX IF NOT EXISTS idx_files_language    ON files(language);
CREATE INDEX IF NOT EXISTS idx_files_created     ON files(created_at);
CREATE INDEX IF NOT EXISTS idx_files_sha256      ON files(sha256);
CREATE INDEX IF NOT EXISTS idx_versions_file     ON versions(file_id);

-- ── FTS5 full-text search ─────────────────────────────────────────────────────
-- Covers files (path + description + body content).
-- Collections are searched via a separate simpler query on the collections table.
CREATE VIRTUAL TABLE IF NOT EXISTS files_fts USING fts5(
    file_id     UNINDEXED,
    path,
    description,
    content,
    tokenize = 'porter unicode61'
);

CREATE TRIGGER IF NOT EXISTS files_fts_insert
    AFTER INSERT ON files BEGIN
        INSERT INTO files_fts(file_id, path, description, content)
        VALUES (new.id, new.path, new.description, '');
    END;

CREATE TRIGGER IF NOT EXISTS files_fts_update
    AFTER UPDATE ON files BEGIN
        UPDATE files_fts
        SET path = new.path, description = new.description
        WHERE file_id = old.id;
    END;

CREATE TRIGGER IF NOT EXISTS files_fts_delete
    BEFORE DELETE ON files BEGIN
        DELETE FROM files_fts WHERE file_id = old.id;
    END;

-- ── blob ref-count triggers ───────────────────────────────────────────────────
-- Automatically maintain blobs.ref_count as files and versions are created/deleted.

-- File inserted: increment ref
CREATE TRIGGER IF NOT EXISTS blobs_file_insert
    AFTER INSERT ON files BEGIN
        UPDATE blobs SET ref_count = ref_count + 1 WHERE sha256 = new.sha256;
    END;

-- File content updated (new sha256): decrement old ref, increment new ref
CREATE TRIGGER IF NOT EXISTS blobs_file_update
    AFTER UPDATE OF sha256 ON files
    WHEN old.sha256 != new.sha256 BEGIN
        UPDATE blobs SET ref_count = ref_count - 1 WHERE sha256 = old.sha256;
        UPDATE blobs SET ref_count = ref_count + 1 WHERE sha256 = new.sha256;
    END;

-- File deleted: decrement ref
CREATE TRIGGER IF NOT EXISTS blobs_file_delete
    AFTER DELETE ON files BEGIN
        UPDATE blobs SET ref_count = ref_count - 1 WHERE sha256 = old.sha256;
    END;

-- Version inserted: increment ref
CREATE TRIGGER IF NOT EXISTS blobs_version_insert
    AFTER INSERT ON versions BEGIN
        UPDATE blobs SET ref_count = ref_count + 1 WHERE sha256 = new.sha256;
    END;

-- Version deleted: decrement ref
CREATE TRIGGER IF NOT EXISTS blobs_version_delete
    AFTER DELETE ON versions BEGIN
        UPDATE blobs SET ref_count = ref_count - 1 WHERE sha256 = old.sha256;
    END;-- schema.sql

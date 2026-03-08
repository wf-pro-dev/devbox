-- devbox schema
-- This file is the single source of truth for the database structure.
-- It is read by sqlc to generate Go code and executed on startup to
-- initialise the database.

-- ── files ──────────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS files (
    id          TEXT    PRIMARY KEY,
    name        TEXT    NOT NULL,
    description TEXT    NOT NULL DEFAULT '',
    language    TEXT    NOT NULL DEFAULT '',
    size        INTEGER NOT NULL DEFAULT 0,
    blob_path   TEXT    NOT NULL,
    sha256      TEXT    NOT NULL,
    uploaded_by TEXT    NOT NULL DEFAULT '',
    created_at  TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now')),
    updated_at  TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now'))
);

-- ── tags ───────────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS tags (
    id   INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT    NOT NULL UNIQUE
);

-- ── file_tags ──────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS file_tags (
    file_id TEXT    NOT NULL REFERENCES files(id) ON DELETE CASCADE,
    tag_id  INTEGER NOT NULL REFERENCES tags(id)  ON DELETE CASCADE,
    PRIMARY KEY (file_id, tag_id)
);

-- ── versions ───────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS versions (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    file_id     TEXT    NOT NULL REFERENCES files(id) ON DELETE CASCADE,
    blob_path   TEXT    NOT NULL,
    sha256      TEXT    NOT NULL,
    size        INTEGER NOT NULL DEFAULT 0,
    uploaded_by TEXT    NOT NULL DEFAULT '',
    created_at  TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now'))
);

-- ── transfers ──────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS transfers (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    from_host   TEXT    NOT NULL,
    to_host     TEXT    NOT NULL,
    filename    TEXT    NOT NULL,
    size        INTEGER NOT NULL DEFAULT 0,
    duration_ms INTEGER NOT NULL DEFAULT 0,
    created_at  TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now'))
);

-- ── indexes ────────────────────────────────────────────────────────────────
CREATE INDEX IF NOT EXISTS idx_files_language  ON files(language);
CREATE INDEX IF NOT EXISTS idx_files_created   ON files(created_at);
CREATE INDEX IF NOT EXISTS idx_versions_file   ON versions(file_id);
CREATE INDEX IF NOT EXISTS idx_transfers_hosts ON transfers(from_host, to_host);

-- ── FTS5 full-text search ──────────────────────────────────────────────────
CREATE VIRTUAL TABLE IF NOT EXISTS files_fts USING fts5(
    file_id UNINDEXED,
    name,
    description,
    content,
    tokenize = 'porter unicode61'
);

CREATE TRIGGER IF NOT EXISTS files_fts_insert
    AFTER INSERT ON files BEGIN
        INSERT INTO files_fts(file_id, name, description, content)
        VALUES (new.id, new.name, new.description, '');
    END;

CREATE TRIGGER IF NOT EXISTS files_fts_update
    AFTER UPDATE ON files BEGIN
        UPDATE files_fts
        SET name = new.name, description = new.description
        WHERE file_id = old.id;
    END;

CREATE TRIGGER IF NOT EXISTS files_fts_delete
    BEFORE DELETE ON files BEGIN
        DELETE FROM files_fts WHERE file_id = old.id;
    END;
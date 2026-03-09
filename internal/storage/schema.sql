-- devbox schema

-- ── directories ──────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS directories (
    id          TEXT    PRIMARY KEY,
    name        TEXT    NOT NULL,
    prefix      TEXT    NOT NULL DEFAULT '',
    description TEXT    NOT NULL DEFAULT '',
    uploaded_by TEXT    NOT NULL DEFAULT '',
    created_at  TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now')),
    updated_at  TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now'))
);

-- ── files ────────────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS files (
    id          TEXT    PRIMARY KEY,
    path        TEXT    NOT NULL,
    file_name   TEXT    NOT NULL DEFAULT '',
    dir_id      TEXT    REFERENCES directories(id) ON DELETE SET NULL,
    dir_prefix  TEXT    NOT NULL DEFAULT '',
    description TEXT    NOT NULL DEFAULT '',
    language    TEXT    NOT NULL DEFAULT '',
    size        INTEGER NOT NULL DEFAULT 0,
    blob_path   TEXT    NOT NULL,
    sha256      TEXT    NOT NULL,
    uploaded_by TEXT    NOT NULL DEFAULT '',
    version     INTEGER NOT NULL DEFAULT 1,      -- current version number
    created_at  TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now')),
    updated_at  TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now'))
);

-- ── tags ─────────────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS tags (
    id   INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT    NOT NULL UNIQUE
);

-- ── file_tags ─────────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS file_tags (
    file_id TEXT    NOT NULL REFERENCES files(id) ON DELETE CASCADE,
    tag_id  INTEGER NOT NULL REFERENCES tags(id)  ON DELETE CASCADE,
    PRIMARY KEY (file_id, tag_id)
);

-- ── versions ──────────────────────────────────────────────────────────────────
-- Each row is a snapshot of a file at a specific version.
-- version_number 1 = original upload, 2 = first update, etc.
CREATE TABLE IF NOT EXISTS versions (
    id             INTEGER PRIMARY KEY AUTOINCREMENT,
    file_id        TEXT    NOT NULL REFERENCES files(id) ON DELETE CASCADE,
    version_number INTEGER NOT NULL DEFAULT 1,
    blob_path      TEXT    NOT NULL,
    sha256         TEXT    NOT NULL,
    size           INTEGER NOT NULL DEFAULT 0,
    uploaded_by    TEXT    NOT NULL DEFAULT '',
    message        TEXT    NOT NULL DEFAULT '',
    created_at     TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now'))
);

-- ── transfers ─────────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS transfers (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    from_host   TEXT    NOT NULL,
    to_host     TEXT    NOT NULL,
    filename    TEXT    NOT NULL,
    size        INTEGER NOT NULL DEFAULT 0,
    duration_ms INTEGER NOT NULL DEFAULT 0,
    created_at  TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now'))
);

-- ── indexes ───────────────────────────────────────────────────────────────────
CREATE INDEX IF NOT EXISTS idx_files_language    ON files(language);
CREATE INDEX IF NOT EXISTS idx_files_created     ON files(created_at);
CREATE INDEX IF NOT EXISTS idx_files_dir         ON files(dir_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_versions_file_num ON versions(file_id, version_number);
CREATE INDEX IF NOT EXISTS idx_versions_file     ON versions(file_id);

-- ── FTS5 full-text search ────────────────────────────────────────────────────
CREATE VIRTUAL TABLE IF NOT EXISTS files_fts USING fts5(
    file_id UNINDEXED,
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
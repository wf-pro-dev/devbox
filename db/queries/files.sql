-- name: CreateFile :one
INSERT INTO files (id, path, file_name, dir_id, dir_prefix, description, language, size, blob_path, sha256, uploaded_by)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: GetFile :one
SELECT * FROM files WHERE id = ? LIMIT 1;

-- name: ListFiles :many
SELECT * FROM files ORDER BY created_at DESC;

-- name: ListFilesByDir :many
SELECT * FROM files WHERE dir_id = ? ORDER BY path ASC;

-- name: ListFileIDsForDir :many
SELECT id FROM files WHERE dir_id = ?;

-- name: ListFilesNoDir :many
SELECT * FROM files WHERE dir_id IS NULL ORDER BY created_at DESC;

-- name: ListFilesByLanguage :many
SELECT * FROM files WHERE language = ? ORDER BY created_at DESC;

-- name: UpdateFile :one
UPDATE files
SET path        = ?,
    file_name   = ?,
    description = ?,
    language    = ?,
    size        = ?,
    blob_path   = ?,
    sha256      = ?,
    uploaded_by = ?,
    updated_at  = strftime('%Y-%m-%dT%H:%M:%SZ','now')
WHERE id = ?
RETURNING *;

-- name: DeleteFile :exec
DELETE FROM files WHERE id = ?;

-- name: UpdateFileContent :one
-- Updates blob content and bumps version number. Called after snapshotting.
UPDATE files
SET blob_path   = ?,
    sha256      = ?,
    size        = ?,
    version     = ?,
    uploaded_by = ?,
    updated_at  = strftime('%Y-%m-%dT%H:%M:%SZ','now')
WHERE id = ?
RETURNING *;
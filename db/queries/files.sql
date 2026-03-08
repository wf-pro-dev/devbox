-- name: CreateFile :one
INSERT INTO files (id, name, description, language, size, blob_path, sha256, uploaded_by)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: GetFile :one
SELECT * FROM files
WHERE id = ?
LIMIT 1;

-- name: ListFiles :many
SELECT * FROM files
ORDER BY created_at DESC;

-- name: ListFilesByLanguage :many
SELECT * FROM files
WHERE language = ?
ORDER BY created_at DESC;

-- name: UpdateFile :one
UPDATE files
SET
    name        = ?,
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
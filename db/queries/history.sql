-- name: CreateVersion :one
INSERT INTO versions (file_id, blob_path, sha256, size, uploaded_by)
VALUES (?, ?, ?, ?, ?)
RETURNING *;

-- name: ListVersionsForFile :many
SELECT * FROM versions
WHERE file_id = ?
ORDER BY created_at DESC;

-- name: CreateTransfer :one
INSERT INTO transfers (from_host, to_host, filename, size, duration_ms)
VALUES (?, ?, ?, ?, ?)
RETURNING *;

-- name: ListTransfers :many
SELECT * FROM transfers
ORDER BY created_at DESC;
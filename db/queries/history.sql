-- name: SnapshotVersion :one
INSERT INTO versions (file_id, version_number, blob_path, sha256, size, uploaded_by, message)
VALUES (?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: GetLatestVersionNumber :one
SELECT COALESCE(MAX(version_number), 0) FROM versions WHERE file_id = ?;

-- name: ListVersionsForFile :many
SELECT * FROM versions
WHERE file_id = ?
ORDER BY version_number DESC;

-- name: CountVersionsForFile :one
SELECT COUNT(*) FROM versions WHERE file_id = ?;

-- name: GetMinVersionToKeep :one
SELECT COALESCE(MIN(version_number), 0)
FROM (
    SELECT version_number FROM versions
    WHERE file_id = ?
    ORDER BY version_number DESC
    LIMIT ?
) AS kept;

-- name: GetOldBlobPaths :many
SELECT blob_path FROM versions
WHERE file_id = ?
  AND version_number < ?;

-- name: PruneOldVersions :exec
DELETE FROM versions
WHERE file_id = ?
  AND version_number < ?;

-- name: CreateTransfer :one
INSERT INTO transfers (from_host, to_host, filename, size, duration_ms)
VALUES (?, ?, ?, ?, ?)
RETURNING *;

-- name: ListTransfers :many
SELECT * FROM transfers ORDER BY created_at DESC;
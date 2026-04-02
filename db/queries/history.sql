-- name: SnapshotVersion :one
INSERT INTO versions (file_id, version, sha256, size, uploaded_by, message)
VALUES (?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: GetLatestVersionNumber :one
SELECT COALESCE(MAX(version), 0) FROM versions WHERE file_id = ?;

-- GetVersion gets a specific version of a file.
-- version: the version number to get
-- name: GetVersion :one
SELECT * FROM versions WHERE file_id = ? AND version = ?;

-- ListVersions is the single entry point for querying versions.
-- Pass NULL for any filter you want to ignore.
-- file_id: versions for a single file
-- prefix : versions for all files whose path starts with prefix
-- Both can be set at once to scope to a file within a prefix.
-- name: ListVersions :many
SELECT v.*
FROM versions v
JOIN files f ON f.id = v.file_id
WHERE (sqlc.narg(file_id) IS NULL OR v.file_id  =    sqlc.narg(file_id))
  AND (sqlc.narg(prefix)  IS NULL OR f.path LIKE sqlc.narg(prefix) || '%')
ORDER BY f.path ASC, v.version DESC;

-- name: CountVersionsForFile :one
SELECT COUNT(*) FROM versions WHERE file_id = ?;

-- name: GetMinVersionToKeep :one
SELECT COALESCE(MIN(version), 0)
FROM (
    SELECT version FROM versions
    WHERE file_id = ?
    ORDER BY version DESC
    LIMIT ?
) AS kept;

-- name: ListPrunableVersions :many
SELECT sha256, version FROM versions
WHERE file_id = ?
  AND version < ?;

-- name: PruneOldVersions :exec
DELETE FROM versions
WHERE file_id = ?
  AND version < ?;

-- name: ListRollbackVersions :many
SELECT * FROM versions
WHERE file_id = ?
  AND version > ?;

-- name: RollbackToVersion :exec
DELETE FROM versions
WHERE file_id = ?
  AND version > ?;


-- name: CreateTransfer :one
INSERT INTO transfers (from_host, to_host, file_path, size, duration_ms)
VALUES (?, ?, ?, ?, ?)
RETURNING *;

-- name: ListTransfers :many
SELECT * FROM transfers ORDER BY created_at DESC LIMIT 100;

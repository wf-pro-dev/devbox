-- name: CreateFile :one
INSERT INTO files (id, path, file_name, description, language, size, sha256, uploaded_by)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: GetFiles :many
SELECT * FROM files
WHERE id IN (sqlc.slice(ids))
ORDER BY path ASC;

-- name: GetFilesByPath :many
SELECT * FROM files
WHERE path IN (sqlc.slice(paths))
ORDER BY path ASC;

-- ListFiles is the single entry point for querying files.
-- Pass NULL for any filter you want to ignore.
-- prefix: path LIKE prefix || '%'   e.g. 'nginx/'
-- tag   : join on tags.name         e.g. 'prod'
-- lang  : language =                e.g. 'bash'
-- name: ListFiles :many
SELECT DISTINCT f.*
FROM files f
LEFT JOIN file_tags ft ON ft.file_id = f.id
LEFT JOIN tags      t  ON t.id = ft.tag_id
WHERE (sqlc.narg(prefix) IS NULL OR f.path     LIKE sqlc.narg(prefix) || '%')
  AND (sqlc.narg(tag)    IS NULL OR t.name    =     sqlc.narg(tag))
  AND (sqlc.narg(lang)   IS NULL OR f.language =    sqlc.narg(lang))
ORDER BY f.path ASC;

-- name: ListDistinctDirs :many
SELECT DISTINCT substr(path, 1, instr(path, '/') - 1) AS dir
FROM files
WHERE instr(path, '/') > 0
ORDER BY dir ASC;

-- name: CountFiles :one
SELECT COUNT(*) FROM files;

-- name: CountFilesByPrefix :one
SELECT COUNT(*) FROM files WHERE path LIKE ? || '%';

-- name: UpdateFileContent :one
UPDATE files
SET sha256 = ?,
    size        = ?,
    version     = ?,
    uploaded_by = ?,
    updated_at  = strftime('%Y-%m-%dT%H:%M:%SZ','now')
WHERE id = ?
RETURNING *;

-- name: UpdateFileMeta :one
UPDATE files
SET description = ?,
    language    = ?,
    updated_at  = strftime('%Y-%m-%dT%H:%M:%SZ','now')
WHERE id = ?
RETURNING *;

-- name: MoveFile :one
UPDATE files
SET path       = ?,
    file_name  = ?,
    updated_at = strftime('%Y-%m-%dT%H:%M:%SZ','now')
WHERE id = ?
RETURNING *;

-- name: DeleteFile :exec
DELETE FROM files WHERE id = ?;

-- name: DeleteFiles :exec
DELETE FROM files WHERE id IN (sqlc.slice(ids));

-- name: DeleteFilesByPrefix :exec
DELETE FROM files WHERE path LIKE ? || '%';
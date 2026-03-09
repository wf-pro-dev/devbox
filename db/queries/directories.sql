-- name: CreateDirectory :one
INSERT INTO directories (id, name, prefix, description, uploaded_by)
VALUES (?, ?, ?, ?, ?)
RETURNING *;

-- name: GetDirectory :one
SELECT * FROM directories WHERE id = ? LIMIT 1;

-- name: ListDirectories :many
SELECT * FROM directories ORDER BY created_at DESC;

-- name: DeleteDirectory :exec
DELETE FROM directories WHERE id = ?;
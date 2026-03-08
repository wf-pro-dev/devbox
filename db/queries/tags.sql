-- name: CreateTag :one
INSERT INTO tags (name)
VALUES (?)
ON CONFLICT(name) DO UPDATE SET name = excluded.name
RETURNING *;

-- name: GetTagByName :one
SELECT * FROM tags WHERE name = ? LIMIT 1;

-- name: ListTagsForFile :many
SELECT t.* FROM tags t
JOIN file_tags ft ON ft.tag_id = t.id
WHERE ft.file_id = ?
ORDER BY t.name;

-- name: ListFilesForTag :many
SELECT f.* FROM files f
JOIN file_tags ft ON ft.file_id = f.id
JOIN tags t ON t.id = ft.tag_id
WHERE t.name = ?
ORDER BY f.created_at DESC;

-- name: AddTagToFile :exec
INSERT INTO file_tags (file_id, tag_id)
VALUES (?, ?)
ON CONFLICT DO NOTHING;

-- name: RemoveTagFromFile :exec
DELETE FROM file_tags
WHERE file_id = ? AND tag_id = ?;

-- name: RemoveAllTagsFromFile :exec
DELETE FROM file_tags WHERE file_id = ?;
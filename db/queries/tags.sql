
-- name: UpsertTag :one
INSERT INTO tags (name)
VALUES (?)
ON CONFLICT(name) DO UPDATE SET name = excluded.name
RETURNING *;

-- name: GetTagByName :one
SELECT * FROM tags WHERE name = ? LIMIT 1;

-- name: ListAllTags :many
SELECT * FROM tags ORDER BY name ASC;

-- ListTagsForFiles returns tags for one or more files in a single query.
-- Returns (file_id, tag id, tag name) so the caller can group by file_id.
-- Pass a slice with a single element to query tags for one file.
-- name: ListTagsForFiles :many
SELECT ft.file_id, t.id, t.name
FROM tags t
JOIN file_tags ft ON ft.tag_id = t.id
WHERE ft.file_id IN (sqlc.slice(ids))
ORDER BY ft.file_id, t.name ASC;

-- name: AddTagToFile :exec
INSERT OR IGNORE INTO file_tags (file_id, tag_id)
VALUES (?, ?);

-- name: AddTagToFilesByPrefix :exec
INSERT OR IGNORE INTO file_tags (file_id, tag_id)
SELECT f.id, ? FROM files f WHERE f.path LIKE ? || '%';

-- name: RemoveTagFromFile :exec
DELETE FROM file_tags
WHERE file_id = ? AND tag_id = ?;

-- name: RemoveTagFromFilesByPrefix :exec
DELETE FROM file_tags
WHERE tag_id = ?
  AND file_id IN (SELECT id FROM files WHERE path LIKE ? || '%');

-- name: RemoveAllTagsFromFile :exec
DELETE FROM file_tags WHERE file_id = ?;

-- name: RemoveAllTagsFromFilesByPrefix :exec
DELETE FROM file_tags
WHERE file_id IN (SELECT id FROM files WHERE path LIKE ? || '%');

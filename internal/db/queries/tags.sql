-- name: ListTags :many
SELECT * FROM tags ORDER BY name;

-- name: GetTagByName :one
SELECT * FROM tags WHERE name = ?;

-- name: InsertTag :one
INSERT INTO tags (name, origin) VALUES (?, ?)
ON CONFLICT(name) DO UPDATE SET origin = excluded.origin
RETURNING *;

-- name: ListTagsForTask :many
SELECT t.name, t.origin
FROM tags t
JOIN task_tags tt ON tt.tag_id = t.id
WHERE tt.task_id = ?
ORDER BY t.name;

-- name: ClearTaskTags :exec
DELETE FROM task_tags WHERE task_id = ?;

-- name: AttachTaskTag :exec
INSERT OR IGNORE INTO task_tags (task_id, tag_id) VALUES (?, ?);

-- name: ListTasksWithTags :many
SELECT t.id AS task_id, g.name AS tag_name, g.origin AS tag_origin
FROM tasks t
JOIN task_tags tt ON tt.task_id = t.id
JOIN tags g ON g.id = tt.tag_id
WHERE t.deleted_at IS NULL
ORDER BY t.id, g.name;

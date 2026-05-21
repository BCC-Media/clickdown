-- name: ListCommentsForTask :many
SELECT * FROM comments
WHERE task_id = ? AND deleted_at IS NULL
ORDER BY COALESCE(clickup_date, local_created_at) ASC, id ASC;

-- name: GetCommentByClickupID :one
SELECT * FROM comments WHERE clickup_id = ?;

-- name: InsertRemoteComment :one
INSERT INTO comments (clickup_id, task_id, author_id, author_username, text, blocks_json, clickup_date, local_created_at, parent_clickup_id)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: UpdateRemoteComment :exec
UPDATE comments
SET author_id = ?, author_username = ?, text = ?, blocks_json = ?, clickup_date = ?, deleted_at = NULL
WHERE clickup_id = ?;

-- name: InsertLocalComment :one
INSERT INTO comments (clickup_id, task_id, author_id, author_username, text, clickup_date, local_created_at, parent_clickup_id)
VALUES (NULL, ?, ?, ?, ?, NULL, ?, ?)
RETURNING *;

-- name: SetCommentClickupID :exec
UPDATE comments SET clickup_id = ?, clickup_date = ? WHERE id = ?;

-- name: SoftDeleteMissingCommentsForTask :exec
UPDATE comments
SET deleted_at = sqlc.arg(deleted_at)
WHERE task_id = sqlc.arg(task_id)
  AND clickup_id IS NOT NULL
  AND clickup_id NOT IN (sqlc.slice('keep'))
  AND deleted_at IS NULL;

-- name: UpsertCommentDirty :exec
INSERT INTO comments_dirty (comment_id, queued_at) VALUES (?, ?)
ON CONFLICT(comment_id) DO UPDATE SET queued_at = excluded.queued_at;

-- name: ListCommentsDirty :many
SELECT cd.comment_id, cd.queued_at, c.task_id, c.text, c.parent_clickup_id
FROM comments_dirty cd
JOIN comments c ON c.id = cd.comment_id
ORDER BY cd.queued_at ASC;

-- name: ClearCommentDirty :exec
DELETE FROM comments_dirty WHERE comment_id = ?;

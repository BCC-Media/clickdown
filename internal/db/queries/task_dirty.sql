-- name: GetTaskDirty :one
SELECT * FROM task_dirty WHERE task_id = ?;

-- name: ListTaskDirty :many
SELECT * FROM task_dirty;

-- name: UpsertTaskDirty :exec
INSERT INTO task_dirty (task_id, title, description, status, queued_at)
VALUES (?, ?, ?, ?, ?)
ON CONFLICT(task_id) DO UPDATE SET
  title       = task_dirty.title       | excluded.title,
  description = task_dirty.description | excluded.description,
  status      = task_dirty.status      | excluded.status,
  queued_at   = excluded.queued_at;

-- name: ClearTaskDirty :exec
DELETE FROM task_dirty WHERE task_id = ?;

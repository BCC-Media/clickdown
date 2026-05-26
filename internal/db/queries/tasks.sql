-- name: GetTask :one
SELECT * FROM tasks WHERE id = ? AND deleted_at IS NULL;

-- name: GetTaskByClickupID :one
SELECT * FROM tasks WHERE clickup_id = ?;

-- name: ListTasks :many
SELECT * FROM tasks WHERE deleted_at IS NULL ORDER BY id;

-- name: ListClickupIDs :many
SELECT clickup_id FROM tasks WHERE deleted_at IS NULL AND clickup_id IS NOT NULL;

-- name: InsertTask :one
INSERT INTO tasks (
  clickup_id, title, description, status, priority, team_id, list_id,
  due_date, clickup_updated_at, local_updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: InsertLocalTask :one
INSERT INTO tasks (
  clickup_id, title, description, status, list_id, local_updated_at
) VALUES (NULL, ?, ?, ?, ?, ?)
RETURNING *;

-- name: MarkTaskCreated :exec
UPDATE tasks
SET clickup_id = ?,
    team_id = ?,
    clickup_updated_at = ?,
    last_pushed_at = ?
WHERE id = ?;

-- name: UpdateTaskFromRemote :exec
UPDATE tasks
SET title = ?,
    description = ?,
    status = ?,
    priority = ?,
    team_id = ?,
    list_id = ?,
    due_date = ?,
    clickup_updated_at = ?,
    deleted_at = NULL
WHERE id = ?;

-- name: PatchTaskTitle :exec
UPDATE tasks SET title = ?, local_updated_at = ? WHERE id = ?;

-- name: PatchTaskDescription :exec
UPDATE tasks SET description = ?, local_updated_at = ? WHERE id = ?;

-- name: PatchTaskStatus :exec
UPDATE tasks SET status = ?, local_updated_at = ? WHERE id = ?;

-- name: MarkTaskPushed :exec
UPDATE tasks SET last_pushed_at = ?, clickup_updated_at = ? WHERE id = ?;

-- name: SoftDeleteTask :exec
UPDATE tasks SET deleted_at = ? WHERE id = ?;

-- name: SoftDeleteMissingTasks :exec
UPDATE tasks SET deleted_at = ?
WHERE deleted_at IS NULL
  AND clickup_id IS NOT NULL
  AND clickup_id NOT IN (sqlc.slice('keep'));

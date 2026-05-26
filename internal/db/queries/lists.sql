-- name: ListLists :many
SELECT * FROM lists ORDER BY name COLLATE NOCASE;

-- name: GetList :one
SELECT * FROM lists WHERE id = ?;

-- name: UpsertList :exec
INSERT INTO lists (id, name, team_id, updated_at)
VALUES (?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
  name = excluded.name,
  team_id = excluded.team_id,
  updated_at = excluded.updated_at;

-- name: ListStatuses :many
SELECT * FROM statuses ORDER BY list_id, orderindex;

-- name: ListStatusesForList :many
SELECT * FROM statuses WHERE list_id = ? ORDER BY orderindex;

-- name: UpsertStatus :exec
INSERT INTO statuses (list_id, name, color, type, orderindex)
VALUES (?, ?, ?, ?, ?)
ON CONFLICT(list_id, name) DO UPDATE SET
  color = excluded.color,
  type = excluded.type,
  orderindex = excluded.orderindex;

-- name: ClearListStatuses :exec
DELETE FROM statuses WHERE list_id = ?;

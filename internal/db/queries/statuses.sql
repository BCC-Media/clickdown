-- name: ListStatuses :many
SELECT * FROM statuses ORDER BY orderindex;

-- name: UpsertStatus :exec
INSERT INTO statuses (name, color, type, orderindex)
VALUES (?, ?, ?, ?)
ON CONFLICT(name) DO UPDATE SET
  color = excluded.color,
  type = excluded.type,
  orderindex = excluded.orderindex;

-- +goose Up
-- +goose StatementBegin
ALTER TABLE tasks ADD COLUMN list_id TEXT;
-- +goose StatementEnd

CREATE INDEX idx_tasks_list_id ON tasks(list_id);

-- ClickUp statuses are list-scoped. Recreate the table with a composite
-- (list_id, name) primary key so the same status name can coexist across
-- lists with potentially different colors, types, and order. Statuses are
-- repopulated on every sync, so dropping rows here is safe.
-- +goose StatementBegin
DROP TABLE statuses;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TABLE statuses (
  list_id     TEXT NOT NULL,
  name        TEXT NOT NULL,
  color       TEXT NOT NULL,
  type        TEXT NOT NULL,
  orderindex  INTEGER NOT NULL,
  PRIMARY KEY (list_id, name)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE statuses;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TABLE statuses (
  name        TEXT PRIMARY KEY,
  color       TEXT NOT NULL,
  type        TEXT NOT NULL,
  orderindex  INTEGER NOT NULL
);
-- +goose StatementEnd

DROP INDEX idx_tasks_list_id;
ALTER TABLE tasks DROP COLUMN list_id;

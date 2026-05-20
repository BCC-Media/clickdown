-- +goose Up
-- +goose StatementBegin
CREATE TABLE tasks (
  id                  INTEGER PRIMARY KEY AUTOINCREMENT,
  clickup_id          TEXT UNIQUE NOT NULL,
  title               TEXT NOT NULL,
  description         TEXT NOT NULL DEFAULT '',
  status              TEXT NOT NULL,
  priority            INTEGER,
  team_id             TEXT,
  clickup_updated_at  INTEGER,
  local_updated_at    INTEGER NOT NULL,
  last_pushed_at      INTEGER,
  deleted_at          INTEGER
);
-- +goose StatementEnd

CREATE INDEX idx_tasks_status ON tasks(status);
CREATE INDEX idx_tasks_priority ON tasks(priority);
CREATE INDEX idx_tasks_deleted_at ON tasks(deleted_at);

-- +goose StatementBegin
CREATE TABLE statuses (
  name        TEXT PRIMARY KEY,
  color       TEXT NOT NULL,
  type        TEXT NOT NULL,
  orderindex  INTEGER NOT NULL
);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TABLE tags (
  id      INTEGER PRIMARY KEY AUTOINCREMENT,
  name    TEXT NOT NULL UNIQUE,
  origin  TEXT NOT NULL DEFAULT 'local'
);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TABLE task_tags (
  task_id INTEGER NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
  tag_id  INTEGER NOT NULL REFERENCES tags(id)  ON DELETE CASCADE,
  PRIMARY KEY (task_id, tag_id)
);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TABLE task_dirty (
  task_id      INTEGER PRIMARY KEY REFERENCES tasks(id) ON DELETE CASCADE,
  title        INTEGER NOT NULL DEFAULT 0,
  description  INTEGER NOT NULL DEFAULT 0,
  status       INTEGER NOT NULL DEFAULT 0,
  queued_at    INTEGER NOT NULL
);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TABLE settings (
  key   TEXT PRIMARY KEY,
  value TEXT NOT NULL
);
-- +goose StatementEnd

-- +goose Down
DROP TABLE settings;
DROP TABLE task_dirty;
DROP TABLE task_tags;
DROP TABLE tags;
DROP TABLE statuses;
DROP INDEX idx_tasks_deleted_at;
DROP INDEX idx_tasks_priority;
DROP INDEX idx_tasks_status;
DROP TABLE tasks;

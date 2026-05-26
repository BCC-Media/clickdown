-- +goose Up
-- Allow locally-created tasks to live in the DB before they're pushed to
-- ClickUp. The original schema required clickup_id NOT NULL; a fresh local
-- task has no ClickUp ID until the sync worker POSTs it and backfills the
-- real one. SQLite can't DROP NOT NULL in place, so rebuild the table.
-- +goose StatementBegin
CREATE TABLE tasks_new (
  id                  INTEGER PRIMARY KEY AUTOINCREMENT,
  clickup_id          TEXT UNIQUE,
  title               TEXT NOT NULL,
  description         TEXT NOT NULL DEFAULT '',
  status              TEXT NOT NULL,
  priority            INTEGER,
  team_id             TEXT,
  clickup_updated_at  INTEGER,
  local_updated_at    INTEGER NOT NULL,
  last_pushed_at      INTEGER,
  deleted_at          INTEGER,
  list_id             TEXT,
  due_date            INTEGER
);
-- +goose StatementEnd

-- +goose StatementBegin
INSERT INTO tasks_new (
  id, clickup_id, title, description, status, priority, team_id,
  clickup_updated_at, local_updated_at, last_pushed_at, deleted_at, list_id, due_date
)
SELECT id, clickup_id, title, description, status, priority, team_id,
       clickup_updated_at, local_updated_at, last_pushed_at, deleted_at, list_id, due_date
FROM tasks;
-- +goose StatementEnd

DROP TABLE tasks;
ALTER TABLE tasks_new RENAME TO tasks;

CREATE INDEX idx_tasks_status ON tasks(status);
CREATE INDEX idx_tasks_priority ON tasks(priority);
CREATE INDEX idx_tasks_deleted_at ON tasks(deleted_at);
CREATE INDEX idx_tasks_list_id ON tasks(list_id);
CREATE INDEX idx_tasks_due_date ON tasks(due_date);

-- Friendly names for the lists we know about. Populated by the sync worker
-- when it sees a new list_id and (lazily) calls GET /list/{id}. Used to back
-- the create-task modal's list dropdown.
-- +goose StatementBegin
CREATE TABLE lists (
  id          TEXT PRIMARY KEY,
  name        TEXT NOT NULL,
  team_id     TEXT,
  updated_at  INTEGER NOT NULL
);
-- +goose StatementEnd

-- +goose Down
DROP TABLE lists;

-- Restore tasks with NOT NULL clickup_id. Rows with NULL clickup_id (pending
-- creates) are dropped on downgrade.
-- +goose StatementBegin
CREATE TABLE tasks_old (
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
  deleted_at          INTEGER,
  list_id             TEXT,
  due_date            INTEGER
);
-- +goose StatementEnd

-- +goose StatementBegin
INSERT INTO tasks_old SELECT * FROM tasks WHERE clickup_id IS NOT NULL;
-- +goose StatementEnd

DROP TABLE tasks;
ALTER TABLE tasks_old RENAME TO tasks;

CREATE INDEX idx_tasks_status ON tasks(status);
CREATE INDEX idx_tasks_priority ON tasks(priority);
CREATE INDEX idx_tasks_deleted_at ON tasks(deleted_at);
CREATE INDEX idx_tasks_list_id ON tasks(list_id);
CREATE INDEX idx_tasks_due_date ON tasks(due_date);

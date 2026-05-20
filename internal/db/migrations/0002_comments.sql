-- +goose Up
-- +goose StatementBegin
CREATE TABLE comments (
  id                INTEGER PRIMARY KEY AUTOINCREMENT,
  clickup_id        TEXT UNIQUE,
  task_id           INTEGER NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
  author_id         TEXT NOT NULL DEFAULT '',
  author_username   TEXT NOT NULL DEFAULT '',
  text              TEXT NOT NULL,
  clickup_date      INTEGER,
  local_created_at  INTEGER NOT NULL,
  deleted_at        INTEGER
);
-- +goose StatementEnd

CREATE INDEX idx_comments_task ON comments(task_id);

-- +goose StatementBegin
CREATE TABLE comments_dirty (
  comment_id INTEGER PRIMARY KEY REFERENCES comments(id) ON DELETE CASCADE,
  queued_at  INTEGER NOT NULL
);
-- +goose StatementEnd

-- +goose Down
DROP TABLE comments_dirty;
DROP INDEX idx_comments_task;
DROP TABLE comments;

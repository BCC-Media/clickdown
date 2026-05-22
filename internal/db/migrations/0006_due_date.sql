-- +goose Up
-- +goose StatementBegin
ALTER TABLE tasks ADD COLUMN due_date INTEGER;
-- +goose StatementEnd

CREATE INDEX idx_tasks_due_date ON tasks(due_date);

-- +goose Down
DROP INDEX idx_tasks_due_date;
ALTER TABLE tasks DROP COLUMN due_date;

-- +goose Up
ALTER TABLE comments ADD COLUMN parent_clickup_id TEXT;
CREATE INDEX idx_comments_parent ON comments(parent_clickup_id);

-- +goose Down
DROP INDEX idx_comments_parent;
ALTER TABLE comments DROP COLUMN parent_clickup_id;

-- +goose Up
ALTER TABLE comments ADD COLUMN blocks_json TEXT;

-- +goose Down
ALTER TABLE comments DROP COLUMN blocks_json;

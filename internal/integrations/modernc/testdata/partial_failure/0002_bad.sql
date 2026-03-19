-- +goose Up
ALTER TABLE nonexistent_table ADD COLUMN foo TEXT;

-- +goose Down
SELECT 1;

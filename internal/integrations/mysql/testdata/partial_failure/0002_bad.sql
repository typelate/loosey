-- +goose Up
ALTER TABLE nonexistent_table ADD COLUMN foo VARCHAR(255);

-- +goose Down
SELECT 1;

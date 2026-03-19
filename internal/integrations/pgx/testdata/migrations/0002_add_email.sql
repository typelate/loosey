-- +goose Up
ALTER TABLE users ADD COLUMN email TEXT;

-- +goose Down
ALTER TABLE users DROP COLUMN email;

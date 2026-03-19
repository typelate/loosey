-- +goose Up
ALTER TABLE users ADD COLUMN email VARCHAR(255);

-- +goose Down
ALTER TABLE users DROP COLUMN email;

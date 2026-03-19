-- +goose NO TRANSACTION
-- +goose Up
CREATE TABLE surviving_table (id BIGSERIAL PRIMARY KEY, name TEXT);
INSERT INTO nonexistent_table VALUES (1);

-- +goose Down
DROP TABLE IF EXISTS surviving_table;

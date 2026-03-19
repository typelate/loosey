-- +goose Up
CREATE TABLE temp_test (id BIGSERIAL PRIMARY KEY, name TEXT NOT NULL);
INSERT INTO no_such_table VALUES (1);

-- +goose Down
DROP TABLE IF EXISTS temp_test;

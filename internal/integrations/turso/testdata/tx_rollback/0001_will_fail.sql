-- +goose Up
CREATE TABLE temp_test (id INTEGER PRIMARY KEY, name TEXT NOT NULL);
INSERT INTO temp_test (id, name) VALUES (1, 'first');
INSERT INTO temp_test (id, name) VALUES (1, 'duplicate_pk');

-- +goose Down
DROP TABLE IF EXISTS temp_test;

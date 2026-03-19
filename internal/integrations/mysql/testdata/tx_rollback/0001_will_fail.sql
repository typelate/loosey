-- +goose Up
CREATE TABLE temp_test (id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY, name VARCHAR(255) NOT NULL);
INSERT INTO temp_test (id, name) VALUES (1, 'first');
INSERT INTO temp_test (id, name) VALUES (1, 'duplicate_pk');

-- +goose Down
DROP TABLE IF EXISTS temp_test;

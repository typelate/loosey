-- +goose Up
CREATE TABLE widgets (id INTEGER PRIMARY KEY, name TEXT NOT NULL);

-- +goose Down
DROP TABLE widgets;

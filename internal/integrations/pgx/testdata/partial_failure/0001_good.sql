-- +goose Up
CREATE TABLE widgets (id BIGSERIAL PRIMARY KEY, name TEXT NOT NULL);

-- +goose Down
DROP TABLE widgets;

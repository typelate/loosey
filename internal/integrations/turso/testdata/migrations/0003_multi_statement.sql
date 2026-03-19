-- +goose Up
CREATE TABLE posts (
    id INTEGER PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id),
    title TEXT NOT NULL,
    body TEXT
);

CREATE INDEX idx_posts_user_id ON posts(user_id);

-- +goose Down
DROP INDEX idx_posts_user_id;
DROP TABLE posts;

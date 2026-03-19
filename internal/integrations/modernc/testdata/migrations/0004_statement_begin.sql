-- +goose Up
-- +goose StatementBegin
CREATE TABLE tags (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL UNIQUE
);
INSERT INTO tags (name) VALUES ('general');
INSERT INTO tags (name) VALUES ('news');
-- +goose StatementEnd

CREATE TABLE post_tags (
    post_id INTEGER NOT NULL REFERENCES posts(id),
    tag_id INTEGER NOT NULL REFERENCES tags(id),
    PRIMARY KEY (post_id, tag_id)
);

-- +goose Down
DROP TABLE post_tags;
DROP TABLE tags;

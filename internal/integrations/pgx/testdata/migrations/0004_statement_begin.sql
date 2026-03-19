-- +goose Up
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION update_timestamp()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

ALTER TABLE posts ADD COLUMN updated_at TIMESTAMPTZ DEFAULT now();

CREATE TRIGGER posts_update_timestamp
    BEFORE UPDATE ON posts
    FOR EACH ROW
    EXECUTE FUNCTION update_timestamp();

-- +goose Down
DROP TRIGGER IF EXISTS posts_update_timestamp ON posts;
ALTER TABLE posts DROP COLUMN IF EXISTS updated_at;
DROP FUNCTION IF EXISTS update_timestamp();

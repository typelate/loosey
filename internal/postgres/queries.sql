-- name: EnsureTable :exec
CREATE TABLE IF NOT EXISTS goose_db_version (
    id         BIGSERIAL PRIMARY KEY,
    version_id BIGINT NOT NULL,
    is_applied BOOLEAN NOT NULL,
    tstamp     TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- name: InsertVersion :exec
INSERT INTO goose_db_version (version_id, is_applied, tstamp)
VALUES ($1, TRUE, now());

-- name: DeleteVersion :exec
DELETE FROM goose_db_version WHERE version_id = $1;

-- name: ListApplied :many
SELECT DISTINCT version_id
FROM goose_db_version
WHERE is_applied = TRUE
ORDER BY version_id;

-- name: LatestVersion :one
SELECT COALESCE(MAX(version_id), 0)::bigint AS version_id
FROM goose_db_version
WHERE is_applied = TRUE;

-- name: EnsureTable :exec
CREATE TABLE IF NOT EXISTS goose_db_version (
    id         BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
    version_id BIGINT NOT NULL,
    is_applied BOOLEAN NOT NULL,
    tstamp     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- name: InsertVersion :exec
INSERT INTO goose_db_version (version_id, is_applied, tstamp)
VALUES (?, 1, NOW());

-- name: DeleteVersion :exec
DELETE FROM goose_db_version WHERE version_id = ?;

-- name: ListApplied :many
SELECT DISTINCT version_id
FROM goose_db_version
WHERE is_applied = 1
ORDER BY version_id;

-- name: LatestVersion :one
SELECT CAST(COALESCE(MAX(version_id), 0) AS SIGNED) AS version_id
FROM goose_db_version
WHERE is_applied = 1;

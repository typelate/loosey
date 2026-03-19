CREATE TABLE IF NOT EXISTS goose_db_version (
    id         BIGSERIAL PRIMARY KEY,
    version_id BIGINT NOT NULL,
    is_applied BOOLEAN NOT NULL,
    tstamp     TIMESTAMPTZ NOT NULL DEFAULT now()
);

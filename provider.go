// Package loosey is a lightweight database migration library compatible with
// goose (https://github.com/pressly/goose) SQL migration files. It depends
// only on the Go standard library.
//
// Create a manager for your database, point it at your migrations directory,
// and call Up:
//
//	m, err := loosey.NewPostgres(db, os.DirFS("migrations"))
//	results, err := m.Up(ctx)
//
// Supported databases: PostgreSQL, SQLite, MySQL/MariaDB, and libsql/Turso.
// Migration files use goose's annotation format (-- +goose Up, -- +goose Down,
// -- +goose StatementBegin/End, -- +goose NO TRANSACTION).
package loosey

import (
	"bytes"
	"context"
	"database/sql"
	"io/fs"
	"log/slog"
	"sync"
)

//go:generate counterfeiter -generate

// Querier is the interface for migration version tracking operations.
// It is implemented by the dialect-specific packages in internal/.
//
// To use a custom version table name (instead of goose_db_version), copy the
// schema.sql and queries.sql files from the appropriate internal/ dialect
// package, replace the table name, generate with sqlc, and pass the result
// to [New]:
//
//	q := mypkg.New(db)
//	m, err := loosey.New(ctx, db, dir, q, (*mypkg.Queries).WithTx)
//
// See custom_querier.md for a full walkthrough.
type Querier interface {
	EnsureTable(ctx context.Context) error
	InsertVersion(ctx context.Context, versionID int64) error
	DeleteVersion(ctx context.Context, versionID int64) error
	ListApplied(ctx context.Context) ([]int64, error)
	LatestVersion(ctx context.Context) (int64, error)
}

// WithTx creates a copy of a Querier bound to a transaction.
type WithTx[Q Querier] func(Q, *sql.Tx) Q

// Option configures a [Manager].
type Option func(*options)

// WithEnv sets environment variables for ENVSUB expansion in migration files.
// Each string should be in KEY=VALUE format.
func WithEnv(env []string) Option {
	return func(o *options) {
		o.env = env
	}
}

// WithLogger sets a structured logger for migration operations.
// Info level logs migration apply/rollback; Debug level logs each statement.
func WithLogger(logger *slog.Logger) Option {
	return func(o *options) {
		o.logger = logger
	}
}

type options struct {
	env    []string
	logger *slog.Logger
}

// Manager applies and rolls back database migrations.
// Create one with [NewPostgres], [NewSQLite3], [NewMySQL], or [NewLibSQL].
type Manager[Q Querier] struct {
	options
	mu         sync.RWMutex
	db         *sql.DB
	q          Q
	withTx     WithTx[Q]
	migrations []*migration
}

// New creates a Manager with a custom dialect. For built-in dialects, use
// [NewPostgres], [NewSQLite3], [NewMySQL], or [NewLibSQL] instead.
//
// The q parameter is a [Querier] bound to db, and withTx creates a
// transaction-bound copy (typically a (*Queries).WithTx method expression).
func New[Q Querier](ctx context.Context, db *sql.DB, dir fs.FS, q Q, withTx WithTx[Q], opts ...Option) (*Manager[Q], error) {
	m := &Manager[Q]{
		db:     db,
		q:      q,
		withTx: withTx,
	}
	for _, opt := range opts {
		opt(&m.options)
	}
	if m.logger == nil {
		m.logger = slog.New(slog.DiscardHandler)
	}

	migrations, err := collectMigrations(dir, m.env)
	if err != nil {
		return nil, err
	}
	m.migrations = migrations

	if err := q.EnsureTable(ctx); err != nil {
		return nil, err
	}

	return m, nil
}

func parseMigration(dir fs.FS, c *migration, env []string) error {
	buf, err := fs.ReadFile(dir, c.source)
	if err != nil {
		return err
	}
	parsed, err := parseSQL(bytes.NewReader(buf), env)
	if err != nil {
		return &ParseError{Source: c.source, Msg: err.Error()}
	}
	c.up = parsed.up
	c.down = parsed.down
	c.useTx = parsed.useTx
	return nil
}

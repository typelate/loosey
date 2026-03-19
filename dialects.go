package loosey

import (
	"context"
	"database/sql"
	"io/fs"

	"github.com/typelate/loosey/internal/libsql"
	"github.com/typelate/loosey/internal/mysql"
	"github.com/typelate/loosey/internal/postgres"
	"github.com/typelate/loosey/internal/sqlite3"
)

// NewPostgres creates a Manager for PostgreSQL databases.
func NewPostgres(ctx context.Context, db *sql.DB, dir fs.FS, opts ...Option) (*Manager[*postgres.Queries], error) {
	return New(ctx, db, dir, postgres.New(db), (*postgres.Queries).WithTx, opts...)
}

// NewSQLite3 creates a Manager for SQLite databases.
func NewSQLite3(ctx context.Context, db *sql.DB, dir fs.FS, opts ...Option) (*Manager[*sqlite3.Queries], error) {
	return New(ctx, db, dir, sqlite3.New(db), (*sqlite3.Queries).WithTx, opts...)
}

// NewMySQL creates a Manager for MySQL/MariaDB databases.
func NewMySQL(ctx context.Context, db *sql.DB, dir fs.FS, opts ...Option) (*Manager[*mysql.Queries], error) {
	return New(ctx, db, dir, mysql.New(db), (*mysql.Queries).WithTx, opts...)
}

// NewLibSQL creates a Manager for libsql databases.
func NewLibSQL(ctx context.Context, db *sql.DB, dir fs.FS, opts ...Option) (*Manager[*libsql.Queries], error) {
	return New(ctx, db, dir, libsql.New(db), (*libsql.Queries).WithTx, opts...)
}

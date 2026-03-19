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
//
// If you are using pgxpool, use [stdlib.OpenDBFromPool] to obtain a *sql.DB:
//
//	package main
//
//	import (
//		"context"
//		"embed"
//		"fmt"
//		"io/fs"
//		"log"
//		"os"
//
//		"github.com/jackc/pgx/v5/pgxpool"
//		"github.com/jackc/pgx/v5/stdlib"
//
//		"github.com/typelate/loosey"
//	)
//
//	//go:embed migrations/*.sql
//	var migrations embed.FS
//
//	func main() {
//		ctx := context.Background()
//		pool, err := pgxpool.New(ctx, os.Getenv("DATABASE_URL"))
//		if err != nil {
//			log.Fatal(err)
//		}
//		defer pool.Close()
//		db := stdlib.OpenDBFromPool(pool)
//		defer db.Close()
//		dir, err := fs.Sub(migrations, "migrations")
//		if err != nil {
//			log.Fatal(err)
//		}
//		m, err := loosey.NewPostgres(ctx, db, dir)
//		if err != nil {
//			log.Fatal(err)
//		}
//		results, err := m.Up(ctx)
//		if err != nil {
//			log.Fatal(err)
//		}
//		for _, r := range results {
//			fmt.Printf("applied %s (%s)\n", r.Source, r.Duration)
//		}
//	}
//
// [stdlib.OpenDBFromPool]: https://pkg.go.dev/github.com/jackc/pgx/v5/stdlib#OpenDBFromPool
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

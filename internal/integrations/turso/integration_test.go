package turso_test

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	_ "turso.tech/database/tursogo"

	"github.com/typelate/loosey"
	"github.com/typelate/loosey/goosey"
)

var migrationsDir = filepath.FromSlash("testdata/migrations")

func openLibSQL(t *testing.T) (*sql.DB, string) {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	db, err := sql.Open("turso", dbPath)
	require.NoError(t, err)
	// The turso engine locks the database file for as long as a connection is
	// open. Keep no idle connections so the goose CLI (a separate process) can
	// access the file between loosey operations.
	db.SetMaxIdleConns(0)
	t.Cleanup(func() { _ = db.Close() })
	return db, dbPath
}

func Test(t *testing.T) {
	db, _ := openLibSQL(t)
	m, err := loosey.NewLibSQL(t.Context(), db, os.DirFS(migrationsDir))
	require.NoError(t, err)
	goosey.Run(t, m)
}

func TestGooseBefore(t *testing.T) {
	db, dbPath := openLibSQL(t)
	cfg := goosey.Compat{Driver: "turso", DBString: "file:" + dbPath, Dir: migrationsDir}
	goosey.GooseBefore(t, func() goosey.Migrator {
		m, err := loosey.NewLibSQL(t.Context(), db, os.DirFS(migrationsDir))
		require.NoError(t, err)
		return m
	}, cfg)
}

func TestUpPartialFailure(t *testing.T) {
	db, _ := openLibSQL(t)
	m, err := loosey.NewLibSQL(t.Context(), db, os.DirFS(filepath.FromSlash("testdata/partial_failure")))
	require.NoError(t, err)
	goosey.UpFailsAtBadMigration(t, m, 1)
}

func TestTransactionRollback(t *testing.T) {
	db, _ := openLibSQL(t)
	m, err := loosey.NewLibSQL(t.Context(), db, os.DirFS(filepath.FromSlash("testdata/tx_rollback")))
	require.NoError(t, err)
	goosey.TransactionRollsBackOnFailure(t, m)
}

func TestGooseAfter(t *testing.T) {
	db, dbPath := openLibSQL(t)
	cfg := goosey.Compat{Driver: "turso", DBString: "file:" + dbPath, Dir: migrationsDir}
	m, err := loosey.NewLibSQL(t.Context(), db, os.DirFS(migrationsDir))
	require.NoError(t, err)
	goosey.GooseAfter(t, m, cfg)
}

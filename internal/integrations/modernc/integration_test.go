package sqlite_test

import (
	"bytes"
	"database/sql"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"

	"github.com/typelate/loosey"
	"github.com/typelate/loosey/goosey"
)

var migrationsDir = filepath.FromSlash("testdata/migrations")

func openSQLite(t *testing.T) (*sql.DB, string) {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	db, err := sql.Open("sqlite", dbPath)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return db, dbPath
}

func Test(t *testing.T) {
	db, _ := openSQLite(t)
	m, err := loosey.NewSQLite3(t.Context(), db, os.DirFS(migrationsDir))
	require.NoError(t, err)
	goosey.Run(t, m)
}

func TestGooseBefore(t *testing.T) {
	db, dbPath := openSQLite(t)
	cfg := goosey.Compat{Driver: "sqlite3", DBString: dbPath, Dir: migrationsDir}
	goosey.GooseBefore(t, func() goosey.Migrator {
		m, err := loosey.NewSQLite3(t.Context(), db, os.DirFS(migrationsDir))
		require.NoError(t, err)
		return m
	}, cfg)
}

func TestUpPartialFailure(t *testing.T) {
	db, _ := openSQLite(t)
	m, err := loosey.NewSQLite3(t.Context(), db, os.DirFS(filepath.FromSlash("testdata/partial_failure")))
	require.NoError(t, err)
	goosey.UpFailsAtBadMigration(t, m, 1)
}

func TestTransactionRollback(t *testing.T) {
	db, _ := openSQLite(t)
	m, err := loosey.NewSQLite3(t.Context(), db, os.DirFS(filepath.FromSlash("testdata/tx_rollback")))
	require.NoError(t, err)
	goosey.TransactionRollsBackOnFailure(t, m)
}

func TestWithLogger(t *testing.T) {
	db, _ := openSQLite(t)
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	m, err := loosey.NewSQLite3(t.Context(), db, os.DirFS(migrationsDir), loosey.WithLogger(logger))
	require.NoError(t, err)

	_, err = m.Up(t.Context())
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "applying migration")
	assert.Contains(t, output, "applied migration")
	assert.Contains(t, output, "executing statement")
}

func TestGooseAfter(t *testing.T) {
	db, dbPath := openSQLite(t)
	cfg := goosey.Compat{Driver: "sqlite3", DBString: dbPath, Dir: migrationsDir}
	m, err := loosey.NewSQLite3(t.Context(), db, os.DirFS(migrationsDir))
	require.NoError(t, err)
	goosey.GooseAfter(t, m, cfg)
}

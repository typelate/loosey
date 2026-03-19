package mysql_test

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	_ "github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	tcmariadb "github.com/testcontainers/testcontainers-go/modules/mariadb"

	"github.com/typelate/loosey"
	"github.com/typelate/loosey/goosey"
)

var (
	migrationsDir   = filepath.FromSlash("testdata/migrations")
	mariadbVersions = []string{"11.4", "11.7"}
)

func startMariaDB(t *testing.T, version string) (db *sql.DB, connStr string) {
	t.Helper()
	ctx := context.Background()
	image := fmt.Sprintf("mariadb:%s", version)
	container, err := tcmariadb.Run(ctx, image,
		tcmariadb.WithDatabase("lucy_test"),
		tcmariadb.WithUsername("lucy"),
		tcmariadb.WithPassword("lucy"),
	)
	testcontainers.CleanupContainer(t, container)
	require.NoError(t, err)
	connStr, err = container.ConnectionString(ctx)
	require.NoError(t, err)
	db, err = sql.Open("mysql", connStr)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return db, connStr
}

func Test(t *testing.T) {
	for _, v := range mariadbVersions {
		t.Run(v, func(t *testing.T) {
			db, _ := startMariaDB(t, v)
			m, err := loosey.NewMySQL(t.Context(), db, os.DirFS(migrationsDir))
			require.NoError(t, err)
			goosey.Run(t, m)
		})
	}
}

func TestGooseBefore(t *testing.T) {
	for _, v := range mariadbVersions {
		t.Run(v, func(t *testing.T) {
			db, connStr := startMariaDB(t, v)
			cfg := goosey.Compat{Driver: "mysql", DBString: connStr, Dir: migrationsDir}
			goosey.GooseBefore(t, func() goosey.Migrator {
				m, err := loosey.NewMySQL(t.Context(), db, os.DirFS(migrationsDir))
				require.NoError(t, err)
				return m
			}, cfg)
		})
	}
}

func TestGooseAfter(t *testing.T) {
	for _, v := range mariadbVersions {
		t.Run(v, func(t *testing.T) {
			db, connStr := startMariaDB(t, v)
			cfg := goosey.Compat{Driver: "mysql", DBString: connStr, Dir: migrationsDir}
			m, err := loosey.NewMySQL(t.Context(), db, os.DirFS(migrationsDir))
			require.NoError(t, err)
			goosey.GooseAfter(t, m, cfg)
		})
	}
}

func TestUpPartialFailure(t *testing.T) {
	db, _ := startMariaDB(t, mariadbVersions[0])
	m, err := loosey.NewMySQL(t.Context(), db, os.DirFS(filepath.FromSlash("testdata/partial_failure")))
	require.NoError(t, err)
	goosey.UpFailsAtBadMigration(t, m, 1)
}

func TestTransactionRollback(t *testing.T) {
	db, _ := startMariaDB(t, mariadbVersions[0])
	m, err := loosey.NewMySQL(t.Context(), db, os.DirFS(filepath.FromSlash("testdata/tx_rollback")))
	require.NoError(t, err)
	goosey.TransactionRollsBackOnFailure(t, m)
}

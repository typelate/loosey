package postgres_test

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/typelate/loosey"
	"github.com/typelate/loosey/goosey"
)

var (
	migrationsDir    = filepath.FromSlash("testdata/migrations")
	postgresVersions = []int{16, 17, 18}
)

func startPostgres(t *testing.T, version int) (db *sql.DB, connStr string) {
	t.Helper()
	ctx := context.Background()
	image := fmt.Sprintf("postgres:%d", version)
	container, err := tcpostgres.Run(ctx, image,
		tcpostgres.WithDatabase("lucy_test"),
		tcpostgres.WithUsername("lucy"),
		tcpostgres.WithPassword("lucy"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").WithOccurrence(2),
		),
	)
	testcontainers.CleanupContainer(t, container)
	require.NoError(t, err)
	connStr, err = container.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)
	db, err = sql.Open("pgx", connStr)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return db, connStr
}

func Test(t *testing.T) {
	for _, v := range postgresVersions {
		t.Run(strconv.Itoa(v), func(t *testing.T) {
			db, _ := startPostgres(t, v)
			m, err := loosey.NewPostgres(t.Context(), db, os.DirFS(migrationsDir))
			require.NoError(t, err)
			goosey.Run(t, m)
		})
	}
}

func TestGooseBefore(t *testing.T) {
	for _, v := range postgresVersions {
		t.Run(strconv.Itoa(v), func(t *testing.T) {
			db, connStr := startPostgres(t, v)
			cfg := goosey.Compat{Driver: "postgres", DBString: connStr, Dir: migrationsDir}
			goosey.GooseBefore(t, func() goosey.Migrator {
				m, err := loosey.NewPostgres(t.Context(), db, os.DirFS(migrationsDir))
				require.NoError(t, err)
				return m
			}, cfg)
		})
	}
}

func TestGooseAfter(t *testing.T) {
	for _, v := range postgresVersions {
		t.Run(strconv.Itoa(v), func(t *testing.T) {
			db, connStr := startPostgres(t, v)
			cfg := goosey.Compat{Driver: "postgres", DBString: connStr, Dir: migrationsDir}
			m, err := loosey.NewPostgres(t.Context(), db, os.DirFS(migrationsDir))
			require.NoError(t, err)
			goosey.GooseAfter(t, m, cfg)
		})
	}
}

func TestUpPartialFailure(t *testing.T) {
	db, _ := startPostgres(t, postgresVersions[0])
	m, err := loosey.NewPostgres(t.Context(), db, os.DirFS(filepath.FromSlash("testdata/partial_failure")))
	require.NoError(t, err)
	goosey.UpFailsAtBadMigration(t, m, 1)
}

func TestTransactionRollback(t *testing.T) {
	db, _ := startPostgres(t, postgresVersions[0])
	m, err := loosey.NewPostgres(t.Context(), db, os.DirFS(filepath.FromSlash("testdata/tx_rollback")))
	require.NoError(t, err)
	goosey.TransactionRollsBackOnFailure(t, m)
}

func TestNoTransactionPartialFailure(t *testing.T) {
	db, _ := startPostgres(t, postgresVersions[0])
	m, err := loosey.NewPostgres(t.Context(), db, os.DirFS(filepath.FromSlash("testdata/no_tx_failure")))
	require.NoError(t, err)
	goosey.NoTransactionPartialFailure(t, m)
}

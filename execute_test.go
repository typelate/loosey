package loosey

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"github.com/typelate/loosey/internal/fake"
)

func assertErrorContains(t *testing.T, err error, substrs ...string) {
	t.Helper()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	msg := err.Error()
	for _, s := range substrs {
		if !strings.Contains(msg, s) {
			t.Errorf("error %q missing substring %q", msg, s)
		}
	}
}

// driverConn combines driver interfaces needed by database/sql to support
// BeginTx and ExecContext without falling back to Prepare.
//
//counterfeiter:generate --fake-name=Conn -o internal/fake/driver_conn.go . driverConn
type driverConn interface {
	driver.Conn
	driver.ConnBeginTx
	driver.ExecerContext
}

//counterfeiter:generate --fake-name=Connector -o internal/fake/driver_connector.go database/sql/driver.Connector
//counterfeiter:generate --fake-name=Tx -o internal/fake/driver_tx.go database/sql/driver.Tx
//counterfeiter:generate --fake-name=Result -o internal/fake/driver_result.go database/sql/driver.Result

//counterfeiter:generate --fake-name=Querier -o internal/fake/querier.go . querier
type querier = Querier

func newTestManager(fakeQ *fake.Querier, migrations ...*migration) *Manager[*fake.Querier] {
	return &Manager[*fake.Querier]{
		q:          fakeQ,
		migrations: migrations,
	}
}

func newTestManagerWithDB(t *testing.T, fakeQ *fake.Querier, conn *fake.Conn, migrations ...*migration) *Manager[*fake.Querier] {
	t.Helper()
	connector := new(fake.Connector)
	connector.ConnectReturns(conn, nil)
	db := sql.OpenDB(connector)
	t.Cleanup(func() { _ = db.Close() })
	return &Manager[*fake.Querier]{
		db:         db,
		q:          fakeQ,
		withTx:     func(q *fake.Querier, _ *sql.Tx) *fake.Querier { return q },
		migrations: migrations,
		options:    options{logger: slog.New(slog.DiscardHandler)},
	}
}

func TestUp_Empty(t *testing.T) {
	m := newTestManager(new(fake.Querier))
	_, err := m.Up(context.Background())
	if !errors.Is(err, ErrNoMigrations) {
		t.Fatalf("got %v, want ErrNoMigrations", err)
	}
}

func TestVersion_Empty(t *testing.T) {
	fq := new(fake.Querier)
	m := newTestManager(fq, &migration{version: 1, name: "a", source: "00001_a.sql", up: []string{"CREATE TABLE a;"}, useTx: true})
	v, err := m.Version(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v != 0 {
		t.Errorf("version = %d, want 0", v)
	}
}

func newFakeConn(t *testing.T, tx *fake.Tx, result *fake.Result) *fake.Conn {
	t.Helper()
	conn := new(fake.Conn)
	conn.BeginTxReturns(tx, nil)
	conn.ExecContextReturns(result, nil)
	return conn
}

func TestUp_AppliesPending(t *testing.T) {
	fq := new(fake.Querier)
	conn := newFakeConn(t, new(fake.Tx), new(fake.Result))
	m := newTestManagerWithDB(t, fq, conn,
		&migration{version: 1, name: "a", source: "00001_a.sql", up: []string{"CREATE TABLE a;"}, useTx: true},
		&migration{version: 2, name: "b", source: "00002_b.sql", up: []string{"CREATE TABLE b;"}, useTx: true},
	)
	results, err := m.Up(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("got %d results, want 2", len(results))
	}
}

func TestUp_BeginTxError(t *testing.T) {
	fq := new(fake.Querier)
	conn := new(fake.Conn)
	conn.BeginTxReturns(nil, errFake)
	m := newTestManagerWithDB(t, fq, conn,
		&migration{version: 1, name: "a", source: "00001_a.sql", up: []string{"CREATE TABLE a;"}, useTx: true},
	)
	results, err := m.Up(context.Background())
	assertErrorContains(t, err, "00001_a.sql")
	var me *MigrationError
	if !errors.As(err, &me) {
		t.Fatalf("expected *MigrationError, got %T", err)
	}
	if unwrapped := errors.Unwrap(me); !errors.Is(unwrapped, errFake) {
		t.Errorf("Unwrap() = %v, want errFake", unwrapped)
	}
	if len(results) != 1 || !errors.Is(results[0].Err, errFake) {
		t.Errorf("got %v, want wrapped errFake", results[0].Err)
	}
	assertErrorContains(t, results[0].Err, "begin transaction")
}

func TestUp_ExecContextError(t *testing.T) {
	fq := new(fake.Querier)
	tx := new(fake.Tx)
	conn := new(fake.Conn)
	conn.BeginTxReturns(tx, nil)
	conn.ExecContextReturns(nil, errFake)
	m := newTestManagerWithDB(t, fq, conn,
		&migration{version: 1, name: "a", source: "00001_a.sql", up: []string{"CREATE TABLE a;"}, useTx: true},
	)
	results, err := m.Up(context.Background())
	assertErrorContains(t, err, "00001_a.sql")
	assertErrorContains(t, results[0].Err, "statement 1 of 1")
	if tx.RollbackCallCount() != 1 {
		t.Errorf("expected Rollback called once, got %d", tx.RollbackCallCount())
	}
}

func TestUp_CommitError(t *testing.T) {
	fq := new(fake.Querier)
	tx := new(fake.Tx)
	tx.CommitReturns(errFake)
	conn := newFakeConn(t, tx, new(fake.Result))
	m := newTestManagerWithDB(t, fq, conn,
		&migration{version: 1, name: "a", source: "00001_a.sql", up: []string{"CREATE TABLE a;"}, useTx: true},
	)
	results, err := m.Up(context.Background())
	assertErrorContains(t, err, "00001_a.sql")
	if !errors.Is(results[0].Err, errFake) {
		t.Errorf("got %v, want wrapped errFake", results[0].Err)
	}
	assertErrorContains(t, results[0].Err, "commit transaction")
}

func TestUp_InsertVersionError(t *testing.T) {
	fq := new(fake.Querier)
	fq.InsertVersionReturns(errFake)
	tx := new(fake.Tx)
	conn := newFakeConn(t, tx, new(fake.Result))
	m := newTestManagerWithDB(t, fq, conn,
		&migration{version: 1, name: "a", source: "00001_a.sql", up: []string{"CREATE TABLE a;"}, useTx: true},
	)
	results, err := m.Up(context.Background())
	assertErrorContains(t, err, "00001_a.sql")
	if tx.RollbackCallCount() != 1 {
		t.Errorf("expected Rollback called once, got %d", tx.RollbackCallCount())
	}
	assertErrorContains(t, results[0].Err, "recording version")
}

func TestDown_RollsBackLatest(t *testing.T) {
	fq := new(fake.Querier)
	fq.LatestVersionReturns(1, nil)
	conn := newFakeConn(t, new(fake.Tx), new(fake.Result))
	m := newTestManagerWithDB(t, fq, conn,
		&migration{version: 1, name: "a", source: "00001_a.sql", up: []string{"CREATE TABLE a;"}, down: []string{"DROP TABLE a;"}, useTx: true},
	)
	results, err := m.Down(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 || results[0].Direction != DirectionDown {
		t.Errorf("got direction %v, want down", results[0].Direction)
	}
}

func TestDown_DeleteVersionError(t *testing.T) {
	fq := new(fake.Querier)
	fq.LatestVersionReturns(1, nil)
	fq.DeleteVersionReturns(errFake)
	tx := new(fake.Tx)
	conn := newFakeConn(t, tx, new(fake.Result))
	m := newTestManagerWithDB(t, fq, conn,
		&migration{version: 1, name: "a", source: "00001_a.sql", up: []string{"CREATE TABLE a;"}, down: []string{"DROP TABLE a;"}, useTx: true},
	)
	results, err := m.Down(context.Background())
	assertErrorContains(t, err, "00001_a.sql")
	if tx.RollbackCallCount() != 1 {
		t.Errorf("expected Rollback called once, got %d", tx.RollbackCallCount())
	}
	assertErrorContains(t, results[0].Err, "recording rollback")
}

func TestUpByOne_AppliesNext(t *testing.T) {
	fq := new(fake.Querier)
	fq.ListAppliedReturns([]int64{1}, nil)
	conn := newFakeConn(t, new(fake.Tx), new(fake.Result))
	m := newTestManagerWithDB(t, fq, conn,
		&migration{version: 1, name: "a", source: "00001_a.sql", up: []string{"SELECT 1;"}, useTx: true},
		&migration{version: 2, name: "b", source: "00002_b.sql", up: []string{"CREATE TABLE b;"}, useTx: true},
	)
	results, err := m.UpByOne(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 || results[0].Version != 2 {
		t.Errorf("got version %d, want 2", results[0].Version)
	}
}

func TestRedo_RollsBackAndReapplies(t *testing.T) {
	fq := new(fake.Querier)
	fq.LatestVersionReturns(1, nil)
	conn := newFakeConn(t, new(fake.Tx), new(fake.Result))
	m := newTestManagerWithDB(t, fq, conn,
		&migration{version: 1, name: "a", source: "00001_a.sql", up: []string{"CREATE TABLE a;"}, down: []string{"DROP TABLE a;"}, useTx: true},
	)
	results, err := m.Redo(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("got %d results, want 2", len(results))
	}
	if results[0].Direction != DirectionDown || results[1].Direction != DirectionUp {
		t.Errorf("directions = %v, %v; want down, up", results[0].Direction, results[1].Direction)
	}
}

func TestUp_NoTx(t *testing.T) {
	fq := new(fake.Querier)
	conn := new(fake.Conn)
	conn.ExecContextReturns(new(fake.Result), nil)
	m := newTestManagerWithDB(t, fq, conn,
		&migration{version: 1, name: "a", source: "00001_a.sql", up: []string{"CREATE TABLE a;"}, useTx: false},
	)
	results, err := m.Up(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("got %d results, want 1", len(results))
	}
	if conn.BeginTxCallCount() != 0 {
		t.Errorf("BeginTx should not be called for no-tx migration")
	}
}

func TestUp_NoTx_ExecError(t *testing.T) {
	fq := new(fake.Querier)
	conn := new(fake.Conn)
	conn.ExecContextReturns(nil, errFake)
	m := newTestManagerWithDB(t, fq, conn,
		&migration{version: 1, name: "a", source: "00001_a.sql", up: []string{"CREATE TABLE a;"}, useTx: false},
	)
	results, err := m.Up(context.Background())
	assertErrorContains(t, err, "00001_a.sql")
	assertErrorContains(t, results[0].Err, "statement 1 of 1")
}

func TestUp_NoTx_InsertVersionError(t *testing.T) {
	fq := new(fake.Querier)
	fq.InsertVersionReturns(errFake)
	conn := new(fake.Conn)
	conn.ExecContextReturns(new(fake.Result), nil)
	m := newTestManagerWithDB(t, fq, conn,
		&migration{version: 1, name: "a", source: "00001_a.sql", up: []string{"CREATE TABLE a;"}, useTx: false},
	)
	results, err := m.Up(context.Background())
	assertErrorContains(t, err, "00001_a.sql")
	assertErrorContains(t, results[0].Err, "recording version")
}

func TestDown_NoTx_DeleteVersionError(t *testing.T) {
	fq := new(fake.Querier)
	fq.LatestVersionReturns(1, nil)
	fq.DeleteVersionReturns(errFake)
	conn := new(fake.Conn)
	conn.ExecContextReturns(new(fake.Result), nil)
	m := newTestManagerWithDB(t, fq, conn,
		&migration{version: 1, name: "a", source: "00001_a.sql", up: []string{"CREATE TABLE a;"}, down: []string{"DROP TABLE a;"}, useTx: false},
	)
	results, err := m.Down(context.Background())
	assertErrorContains(t, err, "00001_a.sql")
	assertErrorContains(t, results[0].Err, "recording rollback")
}

func TestUp_Idempotent(t *testing.T) {
	fq := new(fake.Querier)
	fq.ListAppliedReturns([]int64{1, 2, 3}, nil)
	m := newTestManager(fq,
		&migration{version: 1, name: "a", source: "00001_a.sql", up: []string{"SELECT 1;"}, useTx: true},
		&migration{version: 2, name: "b", source: "00002_b.sql", up: []string{"SELECT 1;"}, useTx: true},
		&migration{version: 3, name: "c", source: "00003_c.sql", up: []string{"SELECT 1;"}, useTx: true},
	)

	results, err := m.Up(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("got %d results, want 0 (idempotent)", len(results))
	}
}

func TestDown_MissingDownSection(t *testing.T) {
	fq := new(fake.Querier)
	fq.LatestVersionReturns(1, nil)
	m := newTestManager(fq,
		&migration{version: 1, name: "a", source: "00001_a.sql", up: []string{"SELECT 1;"}, useTx: true},
	)

	_, err := m.Down(context.Background())
	if !errors.Is(err, ErrNoDownMigration) {
		t.Fatalf("got %v, want ErrNoDownMigration", err)
	}
	assertErrorContains(t, err, "version 1", "00001_a.sql")
}

func TestUp_OutOfOrder(t *testing.T) {
	fq := new(fake.Querier)
	fq.ListAppliedReturns([]int64{5}, nil)
	m := newTestManager(fq,
		&migration{version: 3, name: "a", source: "00003_a.sql", up: []string{"SELECT 1;"}, useTx: true},
		&migration{version: 5, name: "b", source: "00005_b.sql", up: []string{"SELECT 1;"}, useTx: true},
	)

	_, err := m.Up(context.Background())
	if !errors.Is(err, ErrOutOfOrder) {
		t.Fatalf("got %v, want ErrOutOfOrder", err)
	}
	assertErrorContains(t, err, "version 3", "5")
}

var errFake = errors.New("fake connection lost")

func TestUp_ConnectionLostDuringListApplied(t *testing.T) {
	fq := new(fake.Querier)
	fq.ListAppliedReturns(nil, errFake)
	m := newTestManager(fq, &migration{version: 1, name: "a", source: "00001_a.sql", up: []string{"SELECT 1;"}})
	_, err := m.Up(context.Background())
	if !errors.Is(err, errFake) {
		t.Fatalf("got %v, want wrapped errFake", err)
	}
	assertErrorContains(t, err, "listing applied versions")
}

func TestDown_ConnectionLostDuringLatestVersion(t *testing.T) {
	fq := new(fake.Querier)
	fq.LatestVersionReturns(0, errFake)
	m := newTestManager(fq, &migration{version: 1, name: "a", source: "00001_a.sql", up: []string{"SELECT 1;"}})
	_, err := m.Down(context.Background())
	if !errors.Is(err, errFake) {
		t.Fatalf("got %v, want wrapped errFake", err)
	}
	assertErrorContains(t, err, "getting latest version")
}

func TestDown_NothingToRollback(t *testing.T) {
	fq := new(fake.Querier)
	m := newTestManager(fq, &migration{version: 1, name: "a", source: "00001_a.sql", up: []string{"SELECT 1;"}})
	_, err := m.Down(context.Background())
	if !errors.Is(err, ErrNoNextMigration) {
		t.Fatalf("got %v, want ErrNoNextMigration", err)
	}
}

func TestDown_VersionNotInFilesystem(t *testing.T) {
	fq := new(fake.Querier)
	fq.LatestVersionReturns(99, nil)
	m := newTestManager(fq, &migration{version: 1, name: "a", source: "00001_a.sql", up: []string{"SELECT 1;"}})
	_, err := m.Down(context.Background())
	if !errors.Is(err, ErrVersionNotFound) {
		t.Fatalf("got %v, want ErrVersionNotFound", err)
	}
	assertErrorContains(t, err, "99")
}

func TestUpByOne_ConnectionLostDuringListApplied(t *testing.T) {
	fq := new(fake.Querier)
	fq.ListAppliedReturns(nil, errFake)
	m := newTestManager(fq, &migration{version: 1, name: "a", source: "00001_a.sql", up: []string{"SELECT 1;"}})
	_, err := m.UpByOne(context.Background())
	if !errors.Is(err, errFake) {
		t.Fatalf("got %v, want wrapped errFake", err)
	}
	assertErrorContains(t, err, "listing applied versions")
}

func TestUpByOne_NothingPending(t *testing.T) {
	fq := new(fake.Querier)
	fq.ListAppliedReturns([]int64{1, 2}, nil)
	m := newTestManager(fq,
		&migration{version: 1, name: "a", source: "00001_a.sql", up: []string{"SELECT 1;"}},
		&migration{version: 2, name: "b", source: "00002_b.sql", up: []string{"SELECT 1;"}},
	)
	_, err := m.UpByOne(context.Background())
	if !errors.Is(err, ErrNoNextMigration) {
		t.Fatalf("got %v, want ErrNoNextMigration", err)
	}
}

func TestUpTo_ConnectionLostDuringListApplied(t *testing.T) {
	fq := new(fake.Querier)
	fq.ListAppliedReturns(nil, errFake)
	m := newTestManager(fq, &migration{version: 1, name: "a", source: "00001_a.sql", up: []string{"SELECT 1;"}})
	_, err := m.UpTo(context.Background(), 1)
	if !errors.Is(err, errFake) {
		t.Fatalf("got %v, want wrapped errFake", err)
	}
	assertErrorContains(t, err, "listing applied versions")
}

func TestUpTo_NothingInRange(t *testing.T) {
	fq := new(fake.Querier)
	fq.ListAppliedReturns([]int64{1}, nil)
	m := newTestManager(fq,
		&migration{version: 1, name: "a", source: "00001_a.sql", up: []string{"SELECT 1;"}},
		&migration{version: 5, name: "b", source: "00005_b.sql", up: []string{"SELECT 1;"}},
	)
	results, err := m.UpTo(context.Background(), 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("got %d results, want 0", len(results))
	}
}

func TestUpTo_AlreadyPastTarget(t *testing.T) {
	fq := new(fake.Querier)
	fq.ListAppliedReturns([]int64{1, 2, 3}, nil)
	m := newTestManager(fq,
		&migration{version: 1, name: "a", source: "00001_a.sql", up: []string{"SELECT 1;"}},
		&migration{version: 2, name: "b", source: "00002_b.sql", up: []string{"SELECT 1;"}},
		&migration{version: 3, name: "c", source: "00003_c.sql", up: []string{"SELECT 1;"}},
	)
	results, err := m.UpTo(context.Background(), 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("got %d results, want 0", len(results))
	}
}

func TestDownTo_ConnectionLostDuringListApplied(t *testing.T) {
	fq := new(fake.Querier)
	fq.ListAppliedReturns(nil, errFake)
	m := newTestManager(fq, &migration{version: 1, name: "a", source: "00001_a.sql", up: []string{"SELECT 1;"}, down: []string{"DROP TABLE a;"}})
	_, err := m.DownTo(context.Background(), 0)
	if !errors.Is(err, errFake) {
		t.Fatalf("got %v, want wrapped errFake", err)
	}
	assertErrorContains(t, err, "listing applied versions")
}

func TestDownTo_AlreadyAtTarget(t *testing.T) {
	fq := new(fake.Querier)
	fq.ListAppliedReturns([]int64{1}, nil)
	m := newTestManager(fq,
		&migration{version: 1, name: "a", source: "00001_a.sql", up: []string{"SELECT 1;"}, down: []string{"DROP TABLE a;"}},
		&migration{version: 2, name: "b", source: "00002_b.sql", up: []string{"SELECT 1;"}, down: []string{"DROP TABLE b;"}},
	)
	results, err := m.DownTo(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("got %d results, want 0", len(results))
	}
}

func TestDownTo_MissingDownSection(t *testing.T) {
	fq := new(fake.Querier)
	fq.ListAppliedReturns([]int64{1, 2}, nil)
	m := newTestManager(fq,
		&migration{version: 1, name: "a", source: "00001_a.sql", up: []string{"SELECT 1;"}, down: []string{"DROP TABLE a;"}},
		&migration{version: 2, name: "b", source: "00002_b.sql", up: []string{"SELECT 1;"}},
	)
	_, err := m.DownTo(context.Background(), 0)
	if !errors.Is(err, ErrNoDownMigration) {
		t.Fatalf("got %v, want ErrNoDownMigration", err)
	}
	assertErrorContains(t, err, "version 2", "00002_b.sql")
}

func TestRedo_ConnectionLostDuringLatestVersion(t *testing.T) {
	fq := new(fake.Querier)
	fq.LatestVersionReturns(0, errFake)
	m := newTestManager(fq, &migration{version: 1, name: "a", source: "00001_a.sql", up: []string{"SELECT 1;"}, down: []string{"DROP TABLE a;"}})
	_, err := m.Redo(context.Background())
	if !errors.Is(err, errFake) {
		t.Fatalf("got %v, want wrapped errFake", err)
	}
	assertErrorContains(t, err, "getting latest version")
}

func TestRedo_NothingToRedo(t *testing.T) {
	fq := new(fake.Querier)
	m := newTestManager(fq, &migration{version: 1, name: "a", source: "00001_a.sql", up: []string{"SELECT 1;"}, down: []string{"DROP TABLE a;"}})
	_, err := m.Redo(context.Background())
	if !errors.Is(err, ErrNoNextMigration) {
		t.Fatalf("got %v, want ErrNoNextMigration", err)
	}
}

func TestRedo_VersionNotInFilesystem(t *testing.T) {
	fq := new(fake.Querier)
	fq.LatestVersionReturns(99, nil)
	m := newTestManager(fq, &migration{version: 1, name: "a", source: "00001_a.sql", up: []string{"SELECT 1;"}, down: []string{"DROP TABLE a;"}})
	_, err := m.Redo(context.Background())
	if !errors.Is(err, ErrVersionNotFound) {
		t.Fatalf("got %v, want ErrVersionNotFound", err)
	}
	assertErrorContains(t, err, "99")
}

func TestRedo_MissingDownSection(t *testing.T) {
	fq := new(fake.Querier)
	fq.LatestVersionReturns(1, nil)
	m := newTestManager(fq, &migration{version: 1, name: "a", source: "00001_a.sql", up: []string{"SELECT 1;"}})
	_, err := m.Redo(context.Background())
	if !errors.Is(err, ErrNoDownMigration) {
		t.Fatalf("got %v, want ErrNoDownMigration", err)
	}
	assertErrorContains(t, err, "version 1", "00001_a.sql")
}

func TestReset_ConnectionLostDuringListApplied(t *testing.T) {
	fq := new(fake.Querier)
	fq.ListAppliedReturns(nil, errFake)
	m := newTestManager(fq, &migration{version: 1, name: "a", source: "00001_a.sql", up: []string{"SELECT 1;"}, down: []string{"DROP TABLE a;"}})
	_, err := m.Reset(context.Background())
	if !errors.Is(err, errFake) {
		t.Fatalf("got %v, want wrapped errFake", err)
	}
	assertErrorContains(t, err, "listing applied versions")
}

func TestReset_MissingDownSection(t *testing.T) {
	fq := new(fake.Querier)
	fq.ListAppliedReturns([]int64{1}, nil)
	m := newTestManager(fq, &migration{version: 1, name: "a", source: "00001_a.sql", up: []string{"SELECT 1;"}})
	_, err := m.Reset(context.Background())
	if !errors.Is(err, ErrNoDownMigration) {
		t.Fatalf("got %v, want ErrNoDownMigration", err)
	}
	assertErrorContains(t, err, "version 1", "00001_a.sql")
}

func TestReset_NothingApplied(t *testing.T) {
	fq := new(fake.Querier)
	m := newTestManager(fq,
		&migration{version: 1, name: "a", source: "00001_a.sql", up: []string{"SELECT 1;"}, down: []string{"DROP TABLE a;"}},
	)
	results, err := m.Reset(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("got %d results, want 0", len(results))
	}
}

func TestStatus_ConnectionLostDuringListApplied(t *testing.T) {
	fq := new(fake.Querier)
	fq.ListAppliedReturns(nil, errFake)
	m := newTestManager(fq, &migration{version: 1, name: "a", source: "00001_a.sql"})
	_, err := m.Status(context.Background())
	if !errors.Is(err, errFake) {
		t.Fatalf("got %v, want wrapped errFake", err)
	}
	assertErrorContains(t, err, "listing applied versions")
}

func TestVersion_ConnectionLost(t *testing.T) {
	fq := new(fake.Querier)
	fq.LatestVersionReturns(0, errFake)
	m := newTestManager(fq, &migration{version: 1, name: "a", source: "00001_a.sql"})
	_, err := m.Version(context.Background())
	if !errors.Is(err, errFake) {
		t.Fatalf("got %v, want wrapped errFake", err)
	}
}

func TestDown_Empty(t *testing.T) {
	m := newTestManager(new(fake.Querier))
	_, err := m.Down(context.Background())
	if !errors.Is(err, ErrNoMigrations) {
		t.Fatalf("got %v, want ErrNoMigrations", err)
	}
}

func TestUpByOne_Empty(t *testing.T) {
	m := newTestManager(new(fake.Querier))
	_, err := m.UpByOne(context.Background())
	if !errors.Is(err, ErrNoMigrations) {
		t.Fatalf("got %v, want ErrNoMigrations", err)
	}
}

func TestUpTo_Empty(t *testing.T) {
	m := newTestManager(new(fake.Querier))
	_, err := m.UpTo(context.Background(), 1)
	if !errors.Is(err, ErrNoMigrations) {
		t.Fatalf("got %v, want ErrNoMigrations", err)
	}
}

func TestStatus(t *testing.T) {
	fq := new(fake.Querier)
	fq.ListAppliedReturns([]int64{1, 2}, nil)
	m := newTestManager(fq,
		&migration{version: 1, name: "a", source: "00001_a.sql"},
		&migration{version: 2, name: "b", source: "00002_b.sql"},
		&migration{version: 3, name: "c", source: "00003_c.sql"},
	)

	statuses, err := m.Status(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(statuses) != 3 {
		t.Fatalf("got %d statuses, want 3", len(statuses))
	}
	if !statuses[0].Applied || !statuses[1].Applied || statuses[2].Applied {
		t.Errorf("expected first 2 applied, third not: %v %v %v", statuses[0].Applied, statuses[1].Applied, statuses[2].Applied)
	}
}

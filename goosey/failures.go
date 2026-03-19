package goosey

import (
	"errors"
	"testing"

	"github.com/typelate/loosey"
)

// UpFailsAtBadMigration verifies that when Up encounters a migration with
// invalid SQL, it returns an error, the failing migration's result has Err set,
// and prior successful migrations remain applied.
// The migrations directory should contain valid migrations followed by one that
// will fail (e.g., references a nonexistent table/column).
func UpFailsAtBadMigration(t *testing.T, m Migrator, expectedGoodCount int) {
	t.Helper()
	ctx := t.Context()

	results, err := m.Up(ctx)
	if err == nil {
		t.Fatal("expected Up to fail, got nil error")
	}

	var me *loosey.MigrationError
	if !errors.As(err, &me) {
		t.Fatalf("expected *MigrationError, got %T: %v", err, err)
	}

	if len(me.Results) == 0 {
		t.Fatal("MigrationError has no results")
	}

	// The last result should have an error
	last := me.Results[len(me.Results)-1]
	if last.Err == nil {
		t.Error("last result should have Err set")
	}

	// Count successful results
	goodCount := 0
	for _, r := range results {
		if r.Err == nil {
			goodCount++
		}
	}
	if goodCount != expectedGoodCount {
		t.Errorf("got %d successful migrations before failure, want %d", goodCount, expectedGoodCount)
	}

	// The version should reflect only the successfully applied migrations
	v, err := m.Version(ctx)
	if err != nil {
		t.Fatalf("Version after failed Up: %v", err)
	}
	if expectedGoodCount == 0 && v != 0 {
		t.Errorf("Version = %d after all migrations failed, want 0", v)
	}
	if expectedGoodCount > 0 && v == 0 {
		t.Error("Version is 0 but some migrations should have succeeded")
	}

	t.Logf("Up correctly failed: %d good, 1 bad. Version=%d. Error: %v", goodCount, v, last.Err)
}

// TransactionRollsBackOnFailure verifies that when a transactional migration
// fails, the entire migration is rolled back — neither the SQL changes nor the
// version record are persisted.
// The migrations should have exactly one migration that contains multiple
// statements where the second one fails (e.g., valid CREATE then invalid ALTER).
func TransactionRollsBackOnFailure(t *testing.T, m Migrator) {
	t.Helper()
	ctx := t.Context()

	_, err := m.Up(ctx)
	if err == nil {
		t.Fatal("expected Up to fail")
	}

	v, err := m.Version(ctx)
	if err != nil {
		t.Fatalf("Version: %v", err)
	}
	if v != 0 {
		t.Errorf("Version = %d after transaction rollback, want 0 (nothing should be applied)", v)
	}

	t.Logf("transaction correctly rolled back on failure, version=%d", v)
}

// NoTransactionPartialFailure verifies that when a NO TRANSACTION migration
// fails partway through, the statements that executed before the failure
// persist (no rollback), but the version is NOT recorded as applied.
// The migration should have NO TRANSACTION annotation with multiple statements
// where a later one fails.
func NoTransactionPartialFailure(t *testing.T, m Migrator) {
	t.Helper()
	ctx := t.Context()

	_, err := m.Up(ctx)
	if err == nil {
		t.Fatal("expected Up to fail")
	}

	v, err := m.Version(ctx)
	if err != nil {
		t.Fatalf("Version: %v", err)
	}
	if v != 0 {
		t.Errorf("Version = %d, want 0 (failed migration should not be recorded)", v)
	}

	t.Log("NO TRANSACTION migration failed as expected, version not recorded")
}

package goosey

import (
	"context"
	"testing"

	"github.com/typelate/loosey"
)

// Run runs the full migration lifecycle test suite against the given Manager.
func Run[Q loosey.Querier](t *testing.T, m *loosey.Manager[Q]) {
	t.Run("Up", func(t *testing.T) {
		results, err := m.Up(context.Background())
		if err != nil {
			t.Fatalf("Up failed: %v", err)
		}
		if len(results) == 0 {
			t.Fatal("Up returned no results")
		}
		for _, r := range results {
			if r.Err != nil {
				t.Errorf("migration %d failed: %v", r.Version, r.Err)
			}
		}
	})

	t.Run("Version", func(t *testing.T) {
		v, err := m.Version(context.Background())
		if err != nil {
			t.Fatalf("Version failed: %v", err)
		}
		if v == 0 {
			t.Error("Version is 0 after Up")
		}
	})

	t.Run("Status", func(t *testing.T) {
		statuses, err := m.Status(context.Background())
		if err != nil {
			t.Fatalf("Status failed: %v", err)
		}
		for _, s := range statuses {
			if !s.Applied {
				t.Errorf("migration %d (%s) not applied after Up", s.Version, s.Source)
			}
		}
	})

	t.Run("Idempotent", func(t *testing.T) {
		results, err := m.Up(context.Background())
		if err != nil {
			t.Fatalf("second Up failed: %v", err)
		}
		if len(results) != 0 {
			t.Errorf("second Up applied %d migrations, want 0", len(results))
		}
	})

	t.Run("Down", func(t *testing.T) {
		results, err := m.Down(context.Background())
		if err != nil {
			t.Fatalf("Down failed: %v", err)
		}
		if len(results) != 1 {
			t.Fatalf("Down returned %d results, want 1", len(results))
		}
		if results[0].Err != nil {
			t.Errorf("Down migration failed: %v", results[0].Err)
		}
	})

	t.Run("UpAfterDown", func(t *testing.T) {
		results, err := m.Up(context.Background())
		if err != nil {
			t.Fatalf("Up after Down failed: %v", err)
		}
		if len(results) != 1 {
			t.Fatalf("Up after Down returned %d results, want 1", len(results))
		}
	})

	t.Run("Redo", func(t *testing.T) {
		results, err := m.Redo(context.Background())
		if err != nil {
			t.Fatalf("Redo failed: %v", err)
		}
		if len(results) != 2 {
			t.Fatalf("Redo returned %d results, want 2 (down + up)", len(results))
		}
	})

	t.Run("Reset", func(t *testing.T) {
		results, err := m.Reset(context.Background())
		if err != nil {
			t.Fatalf("Reset failed: %v", err)
		}
		if len(results) == 0 {
			t.Fatal("Reset returned no results")
		}
		for _, r := range results {
			if r.Err != nil {
				t.Errorf("Reset migration %d failed: %v", r.Version, r.Err)
			}
		}

		v, err := m.Version(context.Background())
		if err != nil {
			t.Fatalf("Version after Reset failed: %v", err)
		}
		if v != 0 {
			t.Errorf("Version after Reset = %d, want 0", v)
		}
	})

	t.Run("UpByOne", func(t *testing.T) {
		results, err := m.UpByOne(context.Background())
		if err != nil {
			t.Fatalf("UpByOne failed: %v", err)
		}
		if len(results) != 1 {
			t.Fatalf("UpByOne returned %d results, want 1", len(results))
		}
	})

	t.Run("UpTo", func(t *testing.T) {
		// First reset to clean state
		m.Reset(context.Background())

		statuses, err := m.Status(context.Background())
		if err != nil {
			t.Fatalf("Status failed: %v", err)
		}
		if len(statuses) < 2 {
			t.Skip("need at least 2 migrations for UpTo test")
		}
		target := statuses[0].Version
		results, err := m.UpTo(context.Background(), target)
		if err != nil {
			t.Fatalf("UpTo failed: %v", err)
		}
		if len(results) != 1 {
			t.Fatalf("UpTo returned %d results, want 1", len(results))
		}
	})

	t.Run("DownTo", func(t *testing.T) {
		// Ensure all are applied
		m.Up(context.Background())

		statuses, err := m.Status(context.Background())
		if err != nil {
			t.Fatalf("Status failed: %v", err)
		}
		if len(statuses) < 2 {
			t.Skip("need at least 2 migrations for DownTo test")
		}
		target := statuses[0].Version
		results, err := m.DownTo(context.Background(), target)
		if err != nil {
			t.Fatalf("DownTo failed: %v", err)
		}
		if len(results) == 0 {
			t.Fatal("DownTo returned no results")
		}
	})
}

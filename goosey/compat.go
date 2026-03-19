package goosey

import (
	"bytes"
	"context"
	"os/exec"
	"strconv"
	"strings"
	"testing"

	"github.com/typelate/loosey"
)

// Migrator is the interface exercised by compatibility tests.
type Migrator interface {
	Up(ctx context.Context) ([]loosey.MigrationResult, error)
	Down(ctx context.Context) ([]loosey.MigrationResult, error)
	Status(ctx context.Context) ([]loosey.MigrationStatus, error)
	Version(ctx context.Context) (int64, error)
}

// Compat holds configuration for running goose CLI commands.
type Compat struct {
	// Driver is the goose driver name (e.g., "postgres", "sqlite3", "mysql").
	Driver string
	// DBString is the goose connection string.
	DBString string
	// Dir is the path to the migrations directory.
	Dir string
	// Env is additional environment variables for the goose process.
	Env []string
}

// GooseUp runs "goose up" against the configured database.
func (c Compat) GooseUp(t *testing.T) {
	t.Helper()
	c.run(t, "up")
}

// GooseDown runs "goose down" against the configured database.
func (c Compat) GooseDown(t *testing.T) {
	t.Helper()
	c.run(t, "down")
}

// GooseVersion runs "goose version" and returns the parsed version number.
func (c Compat) GooseVersion(t *testing.T) int64 {
	t.Helper()
	out := c.run(t, "version")
	return parseGooseVersion(out)
}

func (c Compat) run(t *testing.T, args ...string) string {
	t.Helper()
	gooseBin, err := exec.LookPath("goose")
	if err != nil {
		t.Skip("goose binary not found in PATH")
	}
	fullArgs := []string{"-dir", c.Dir, c.Driver, c.DBString}
	fullArgs = append(fullArgs, args...)
	cmd := exec.Command(gooseBin, fullArgs...)
	cmd.Env = append(cmd.Environ(), c.Env...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("goose %s failed: %v\nstdout: %s\nstderr: %s",
			strings.Join(args, " "), err, stdout.String(), stderr.String())
	}
	return stdout.String() + stderr.String()
}

func parseGooseVersion(output string) int64 {
	fields := strings.Fields(output)
	for i := len(fields) - 1; i >= 0; i-- {
		if v, err := strconv.ParseInt(fields[i], 10, 64); err == nil {
			return v
		}
	}
	return 0
}

// AssertVersionsAgree checks that goose version, loosey Version, and querier LatestVersion all match.
func AssertVersionsAgree(t *testing.T, m Migrator, cfg Compat) {
	t.Helper()
	ctx := t.Context()

	gooseV := cfg.GooseVersion(t)
	looseyV, err := m.Version(ctx)
	if err != nil {
		t.Fatalf("loosey Version: %v", err)
	}
	if gooseV != looseyV {
		t.Errorf("version mismatch: goose=%d loosey=%d", gooseV, looseyV)
	}
}

// AssertAllApplied checks that loosey reports all migrations as applied.
func AssertAllApplied(t *testing.T, m Migrator) {
	t.Helper()
	statuses, err := m.Status(t.Context())
	if err != nil {
		t.Fatalf("loosey Status: %v", err)
	}
	for _, s := range statuses {
		if !s.Applied {
			t.Errorf("migration %d (%s) not applied", s.Version, s.Source)
		}
	}
}

// GooseBefore tests loosey interacting with a database where goose applied migrations first.
// newMigrator is called after goose has set up the table to avoid EnsureTable conflicts.
// The database should be clean when called.
func GooseBefore(t *testing.T, newMigrator func() Migrator, cfg Compat) {
	t.Helper()
	ctx := t.Context()

	t.Log("goose up")
	cfg.GooseUp(t)

	m := newMigrator()

	t.Log("loosey reads all applied")
	AssertAllApplied(t, m)
	AssertVersionsAgree(t, m, cfg)

	t.Log("goose down")
	cfg.GooseDown(t)

	t.Log("loosey sees pending")
	AssertHasPending(t, m)
	AssertVersionsAgree(t, m, cfg)

	t.Log("loosey reapplies")
	_, err := m.Up(ctx)
	if err != nil {
		t.Fatalf("loosey Up: %v", err)
	}
	AssertAllApplied(t, m)
	AssertVersionsAgree(t, m, cfg)
}

// GooseAfter tests goose interacting with a database where loosey applied migrations first.
// The database should be clean when called.
func GooseAfter(t *testing.T, m Migrator, cfg Compat) {
	t.Helper()
	ctx := t.Context()

	t.Log("loosey up")
	_, err := m.Up(ctx)
	if err != nil {
		t.Fatalf("loosey Up: %v", err)
	}
	AssertAllApplied(t, m)

	t.Log("goose reads all")
	AssertVersionsAgree(t, m, cfg)

	t.Log("loosey down")
	_, err = m.Down(ctx)
	if err != nil {
		t.Fatalf("loosey Down: %v", err)
	}
	AssertVersionsAgree(t, m, cfg)

	t.Log("goose reapplies")
	cfg.GooseUp(t)
	AssertAllApplied(t, m)
	AssertVersionsAgree(t, m, cfg)
}

// AssertHasPending checks that loosey reports at least one pending migration.
func AssertHasPending(t *testing.T, m Migrator) {
	t.Helper()
	statuses, err := m.Status(t.Context())
	if err != nil {
		t.Fatalf("loosey Status: %v", err)
	}
	for _, s := range statuses {
		if !s.Applied {
			return
		}
	}
	t.Error("expected at least one pending migration")
}

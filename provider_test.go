package loosey

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/typelate/loosey/internal/fake"
)

func TestNewManager(t *testing.T) {
	fq := new(fake.Querier)
	dir := fstest.MapFS{
		"00001_init.sql": &fstest.MapFile{Data: []byte("-- +goose Up\nCREATE TABLE t (id INT);\n")},
	}

	m, err := New(context.Background(), nil, dir, fq, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m == nil {
		t.Fatal("manager is nil")
	}
	if len(m.migrations) != 1 {
		t.Fatalf("got %d migrations, want 1", len(m.migrations))
	}
	if fq.EnsureTableCallCount() != 1 {
		t.Errorf("EnsureTable called %d times, want 1", fq.EnsureTableCallCount())
	}
}

func TestNewManager_ParseError(t *testing.T) {
	fq := new(fake.Querier)
	dir := fstest.MapFS{
		"00001_bad.sql": &fstest.MapFile{Data: []byte("no annotations here")},
	}

	_, err := New(context.Background(), nil, dir, fq, nil)
	if err == nil {
		t.Fatal("expected error for bad SQL, got nil")
	}
	var pe *ParseError
	if !errors.As(err, &pe) {
		t.Fatalf("expected *ParseError, got %T", err)
	}
	if !strings.Contains(pe.Error(), "00001_bad.sql") {
		t.Errorf("ParseError.Error() = %q, want source file in message", pe.Error())
	}
}

func TestNewManager_EnsureTableError(t *testing.T) {
	fq := new(fake.Querier)
	fq.EnsureTableReturns(fmt.Errorf("db error"))
	dir := fstest.MapFS{
		"00001_init.sql": &fstest.MapFile{Data: []byte("-- +goose Up\nCREATE TABLE t (id INT);\n")},
	}

	_, err := New(context.Background(), nil, dir, fq, nil)
	if err == nil {
		t.Fatal("expected error when EnsureTable fails, got nil")
	}
}

func TestNewManager_WithEnv(t *testing.T) {
	fq := new(fake.Querier)
	dir := fstest.MapFS{
		"00001_init.sql": &fstest.MapFile{Data: []byte("-- +goose Up\n-- +goose ENVSUB ON\nCREATE TABLE ${TBL} (id INT);\n")},
	}

	m, err := New(context.Background(), nil, dir, fq, nil, WithEnv([]string{"TBL=users"}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(m.migrations) != 1 {
		t.Fatalf("got %d migrations, want 1", len(m.migrations))
	}
}

func TestSchemaIsSubstringOfQueries(t *testing.T) {
	schemaFiles, err := filepath.Glob(filepath.FromSlash("internal/*/schema.sql"))
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range schemaFiles {
		pkg := filepath.Dir(p)
		t.Run(pkg, func(t *testing.T) {
			schema, err := os.ReadFile(filepath.Join(pkg, "schema.sql"))
			if err != nil {
				t.Fatal(err)
			}
			queries, err := os.ReadFile(filepath.Join(pkg, "queries.sql"))
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(queries), string(schema)) {
				t.Errorf("schema.sql is not a substring of queries.sql for %s", p)
			}
		})
	}
}

func TestNewManager_EnvsubNoEnv(t *testing.T) {
	fq := new(fake.Querier)
	dir := fstest.MapFS{
		"00001_init.sql": &fstest.MapFile{Data: []byte("-- +goose Up\n-- +goose ENVSUB ON\nCREATE TABLE ${TBL} (id INT);\n")},
	}

	_, err := New(context.Background(), nil, dir, fq, nil)
	if err == nil {
		t.Fatal("expected error when ENVSUB ON without WithEnv, got nil")
	}
}

package loosey

import (
	"testing"
	"testing/fstest"
)

func TestParseMigrationFilename_Sequential(t *testing.T) {
	version, name, err := parseMigrationFilename("00001_create_users.sql")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if version != 1 {
		t.Errorf("version = %d, want 1", version)
	}
	if name != "create_users" {
		t.Errorf("name = %q, want %q", name, "create_users")
	}
}

func TestCollectMigrations(t *testing.T) {
	dir := fstest.MapFS{
		"00003_third.sql":  &fstest.MapFile{Data: []byte("-- +goose Up\n")},
		"00001_first.sql":  &fstest.MapFile{Data: []byte("-- +goose Up\n")},
		"00002_second.sql": &fstest.MapFile{Data: []byte("-- +goose Up\n")},
	}
	migrations, err := collectMigrations(dir, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(migrations) != 3 {
		t.Fatalf("got %d migrations, want 3", len(migrations))
	}
	if migrations[0].version != 1 || migrations[1].version != 2 || migrations[2].version != 3 {
		t.Errorf("migrations not sorted: %d, %d, %d", migrations[0].version, migrations[1].version, migrations[2].version)
	}
}

func TestCollectMigrations_SkipsNonSQL(t *testing.T) {
	dir := fstest.MapFS{
		"00001_first.sql":  &fstest.MapFile{Data: []byte("-- +goose Up\n")},
		"readme.txt":       &fstest.MapFile{Data: []byte("not a migration")},
		"00002_second.sql": &fstest.MapFile{Data: []byte("-- +goose Up\n")},
		".gitkeep":         &fstest.MapFile{Data: []byte("")},
	}
	migrations, err := collectMigrations(dir, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(migrations) != 2 {
		t.Fatalf("got %d migrations, want 2", len(migrations))
	}
}

func TestCollectMigrations_DuplicateVersions(t *testing.T) {
	dir := fstest.MapFS{
		"00001_a.sql": &fstest.MapFile{Data: []byte("-- +goose Up\n")},
		"00001_b.sql": &fstest.MapFile{Data: []byte("-- +goose Up\n")},
	}
	_, err := collectMigrations(dir, nil)
	if err == nil {
		t.Fatal("expected error for duplicate versions, got nil")
	}
}

func TestParseMigrationFilename_Invalid(t *testing.T) {
	invalid := []string{
		"not_a_migration.txt",
		"00001.sql",
		".sql",
		"abc_name.sql",
		"",
		"00001_name.SQL",
	}
	for _, name := range invalid {
		_, _, err := parseMigrationFilename(name)
		if err == nil {
			t.Errorf("expected error for %q, got nil", name)
		}
	}
}

func TestParseMigrationFilename_Timestamp(t *testing.T) {
	version, name, err := parseMigrationFilename("20230115120000_add_index.sql")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if version != 20230115120000 {
		t.Errorf("version = %d, want 20230115120000", version)
	}
	if name != "add_index" {
		t.Errorf("name = %q, want %q", name, "add_index")
	}
}

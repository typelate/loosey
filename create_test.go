package loosey

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func testRoot(t *testing.T) (*os.Root, string) {
	t.Helper()
	dir := t.TempDir()
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatalf("OpenRoot: %v", err)
	}
	t.Cleanup(func() { _ = root.Close() })
	return root, dir
}

func TestCreateMigrationSequential_Empty(t *testing.T) {
	root, dir := testRoot(t)

	filename, err := CreateMigrationSequential(root, "add_users")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if filename != "00001_add_users.sql" {
		t.Errorf("filename = %q, want %q", filename, "00001_add_users.sql")
	}

	content, err := os.ReadFile(filepath.Join(dir, filename))
	if err != nil {
		t.Fatalf("reading created file: %v", err)
	}
	if !strings.Contains(string(content), "-- +goose Up") {
		t.Error("file missing '-- +goose Up'")
	}
	if !strings.Contains(string(content), "-- +goose Down") {
		t.Error("file missing '-- +goose Down'")
	}
}

func TestCreateMigrationSequential_Existing(t *testing.T) {
	root, dir := testRoot(t)
	_ = os.WriteFile(filepath.Join(dir, "00002_existing.sql"), []byte(""), 0o644)

	filename, err := CreateMigrationSequential(root, "next")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if filename != "00003_next.sql" {
		t.Errorf("filename = %q, want %q", filename, "00003_next.sql")
	}
}

func TestCreateMigrationTimestamp(t *testing.T) {
	root, _ := testRoot(t)

	fixed := func() time.Time {
		return time.Date(2025, 3, 15, 14, 30, 45, 0, time.UTC)
	}

	filename, err := CreateMigrationTimestamp(root, "add_index", fixed)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if filename != "20250315143045_add_index.sql" {
		t.Errorf("filename = %q, want %q", filename, "20250315143045_add_index.sql")
	}
}

func TestCreateMigrationTimestamp_NilNow(t *testing.T) {
	root, _ := testRoot(t)

	filename, err := CreateMigrationTimestamp(root, "something", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasSuffix(filename, "_something.sql") {
		t.Errorf("filename = %q, want it to end with _something.sql", filename)
	}
}

func TestSanitizeName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Add User Emails!", "add_user_emails"},
		{"hello-world", "hello_world"},
		{"CamelCase", "camelcase"},
		{"  spaces  ", "spaces"},
		{"a__b", "a_b"},
	}
	for _, tt := range tests {
		got := sanitizeName(tt.input)
		if got != tt.want {
			t.Errorf("sanitizeName(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestCreateMigrationSequential_EmptyName(t *testing.T) {
	root, _ := testRoot(t)

	_, err := CreateMigrationSequential(root, "!!!")
	if err == nil {
		t.Fatal("expected error for empty sanitized name")
	}
	assertErrorContains(t, err, "empty after sanitization")
}

package loosey

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"
	"unicode"
)

const migrationSkeleton = "-- +goose Up\n\n-- +goose Down\n"

// CreateMigrationSequential creates a new SQL migration file with a sequential
// version number. It scans the directory for existing migration files
// and increments from the highest found version.
func CreateMigrationSequential(dir *os.Root, name string) (string, error) {
	sanitized := sanitizeName(name)
	if sanitized == "" {
		return "", fmt.Errorf("loosey: migration name is empty after sanitization")
	}

	var maxVersion int64
	for filename := range listMigrationFiles(dir) {
		version, _, err := parseMigrationFilename(filename)
		if err != nil {
			continue
		}
		if version > maxVersion {
			maxVersion = version
		}
	}

	version := maxVersion + 1
	filename := fmt.Sprintf("%05d_%s.sql", version, sanitized)

	if err := createFile(dir, filename, migrationSkeleton); err != nil {
		return "", fmt.Errorf("loosey: creating migration file: %w", err)
	}
	return filename, nil
}

// CreateMigrationTimestamp creates a new SQL migration file with a timestamp
// version. If now is nil, time.Now is used.
func CreateMigrationTimestamp(dir *os.Root, name string, now func() time.Time) (string, error) {
	sanitized := sanitizeName(name)
	if sanitized == "" {
		return "", fmt.Errorf("loosey: migration name is empty after sanitization")
	}

	if now == nil {
		now = time.Now
	}
	version := now().UTC().Format("20060102150405")
	filename := fmt.Sprintf("%s_%s.sql", version, sanitized)

	if err := createFile(dir, filename, migrationSkeleton); err != nil {
		return "", fmt.Errorf("loosey: creating migration file: %w", err)
	}
	return filename, nil
}

func listMigrationFiles(dir *os.Root) func(yield func(string) bool) {
	return func(yield func(string) bool) {
		f, err := dir.Open(".")
		if err != nil {
			return
		}
		defer func() { _ = f.Close() }()

		entries, err := f.ReadDir(-1)
		if err != nil {
			return
		}
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			if !yield(entry.Name()) {
				return
			}
		}
	}
}

func createFile(dir *os.Root, name, data string) error {
	f, err := dir.Create(name)
	if err != nil {
		return err
	}
	_, writeErr := io.WriteString(f, data)
	closeErr := f.Close()
	if writeErr != nil {
		return writeErr
	}
	return closeErr
}

func sanitizeName(name string) string {
	name = strings.ToLower(name)
	name = strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return r
		}
		return '_'
	}, name)
	for strings.Contains(name, "__") {
		name = strings.ReplaceAll(name, "__", "_")
	}
	name = strings.Trim(name, "_")
	return name
}

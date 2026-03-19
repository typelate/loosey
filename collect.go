package loosey

import (
	"cmp"
	"fmt"
	"io/fs"
	"regexp"
	"slices"
	"strconv"
)

var migrationFileRegex = regexp.MustCompile(`^(\d+)_(.+)\.sql$`)

func parseMigrationFilename(filename string) (int64, string, error) {
	matches := migrationFileRegex.FindStringSubmatch(filename)
	if matches == nil {
		return 0, "", fmt.Errorf("loosey: invalid migration filename: %q", filename)
	}
	version, err := strconv.ParseInt(matches[1], 10, 64)
	if err != nil {
		return 0, "", fmt.Errorf("loosey: invalid version in filename %q: %w", filename, err)
	}
	return version, matches[2], nil
}

type migration struct {
	version int64
	name    string
	source  string
	up      []string
	down    []string
	useTx   bool
}

func collectMigrations(dir fs.FS, env []string) ([]*migration, error) {
	entries, err := fs.ReadDir(dir, ".")
	if err != nil {
		return nil, fmt.Errorf("loosey: reading migrations directory: %w", err)
	}
	var migrations []*migration
	seen := make(map[int64]string)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if len(name) < 5 || name[len(name)-4:] != ".sql" {
			continue
		}
		version, migName, err := parseMigrationFilename(name)
		if err != nil {
			continue
		}
		if existing, ok := seen[version]; ok {
			return nil, fmt.Errorf("loosey: duplicate migration version %d: %q and %q", version, existing, name)
		}
		seen[version] = name
		m := &migration{
			version: version,
			name:    migName,
			source:  name,
		}
		if err := parseMigration(dir, m, env); err != nil {
			return nil, err
		}
		migrations = append(migrations, m)
	}
	slices.SortFunc(migrations, func(a, b *migration) int {
		return cmp.Compare(a.version, b.version)
	})
	return migrations, nil
}

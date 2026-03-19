package loosey

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

type MigrationStatus struct {
	Version   int64
	Name      string
	Source    string
	Applied   bool
	AppliedAt time.Time
}

// Up applies all pending migrations in version order.
func (m *Manager[Q]) Up(ctx context.Context) ([]MigrationResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.migrations) == 0 {
		return nil, ErrNoMigrations
	}

	applied, err := m.q.ListApplied(ctx)
	if err != nil {
		return nil, fmt.Errorf("loosey: listing applied versions: %w", err)
	}
	appliedSet := make(map[int64]bool, len(applied))
	for _, v := range applied {
		appliedSet[v] = true
	}

	var maxApplied int64
	for _, v := range applied {
		if v > maxApplied {
			maxApplied = v
		}
	}

	var pending []*migration
	for _, mig := range m.migrations {
		if !appliedSet[mig.version] {
			pending = append(pending, mig)
		}
	}

	if len(pending) == 0 {
		return nil, nil
	}

	for _, mig := range pending {
		if mig.version < maxApplied {
			return nil, fmt.Errorf("%w: version %d is pending but %d is already applied", ErrOutOfOrder, mig.version, maxApplied)
		}
	}

	var results []MigrationResult
	for _, mig := range pending {
		result := m.executeMigration(ctx, mig, DirectionUp)
		results = append(results, result)
		if result.Err != nil {
			return results, &MigrationError{Results: results}
		}
	}
	return results, nil
}

// Down rolls back the latest applied migration.
func (m *Manager[Q]) Down(ctx context.Context) ([]MigrationResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.migrations) == 0 {
		return nil, ErrNoMigrations
	}

	latest, err := m.q.LatestVersion(ctx)
	if err != nil {
		return nil, fmt.Errorf("loosey: getting latest version: %w", err)
	}
	if latest == 0 {
		return nil, ErrNoNextMigration
	}

	mig := m.findMigration(latest)
	if mig == nil {
		return nil, fmt.Errorf("%w: %d", ErrVersionNotFound, latest)
	}
	if len(mig.down) == 0 {
		return nil, fmt.Errorf("%w: version %d (%s)", ErrNoDownMigration, mig.version, mig.source)
	}

	result := m.executeMigration(ctx, mig, DirectionDown)
	results := []MigrationResult{result}
	if result.Err != nil {
		return results, &MigrationError{Results: results}
	}
	return results, nil
}

// UpByOne applies the next pending migration.
func (m *Manager[Q]) UpByOne(ctx context.Context) ([]MigrationResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.migrations) == 0 {
		return nil, ErrNoMigrations
	}

	applied, err := m.q.ListApplied(ctx)
	if err != nil {
		return nil, fmt.Errorf("loosey: listing applied versions: %w", err)
	}
	appliedSet := make(map[int64]bool, len(applied))
	for _, v := range applied {
		appliedSet[v] = true
	}

	for _, mig := range m.migrations {
		if !appliedSet[mig.version] {
			result := m.executeMigration(ctx, mig, DirectionUp)
			results := []MigrationResult{result}
			if result.Err != nil {
				return results, &MigrationError{Results: results}
			}
			return results, nil
		}
	}

	return nil, ErrNoNextMigration
}

// UpTo applies all pending migrations up to and including the given version.
func (m *Manager[Q]) UpTo(ctx context.Context, version int64) ([]MigrationResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.migrations) == 0 {
		return nil, ErrNoMigrations
	}

	applied, err := m.q.ListApplied(ctx)
	if err != nil {
		return nil, fmt.Errorf("loosey: listing applied versions: %w", err)
	}
	appliedSet := make(map[int64]bool, len(applied))
	for _, v := range applied {
		appliedSet[v] = true
	}

	var results []MigrationResult
	for _, mig := range m.migrations {
		if mig.version > version {
			break
		}
		if appliedSet[mig.version] {
			continue
		}
		result := m.executeMigration(ctx, mig, DirectionUp)
		results = append(results, result)
		if result.Err != nil {
			return results, &MigrationError{Results: results}
		}
	}
	return results, nil
}

// DownTo rolls back all applied migrations down to (but not including) the given version.
func (m *Manager[Q]) DownTo(ctx context.Context, version int64) ([]MigrationResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	applied, err := m.q.ListApplied(ctx)
	if err != nil {
		return nil, fmt.Errorf("loosey: listing applied versions: %w", err)
	}
	appliedSet := make(map[int64]bool, len(applied))
	for _, v := range applied {
		appliedSet[v] = true
	}

	var results []MigrationResult
	for i := len(m.migrations) - 1; i >= 0; i-- {
		mig := m.migrations[i]
		if mig.version <= version || !appliedSet[mig.version] {
			continue
		}
		if len(mig.down) == 0 {
			return results, fmt.Errorf("%w: version %d (%s)", ErrNoDownMigration, mig.version, mig.source)
		}
		result := m.executeMigration(ctx, mig, DirectionDown)
		results = append(results, result)
		if result.Err != nil {
			return results, &MigrationError{Results: results}
		}
	}
	return results, nil
}

// Redo rolls back and re-applies the latest migration.
func (m *Manager[Q]) Redo(ctx context.Context) ([]MigrationResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	latest, err := m.q.LatestVersion(ctx)
	if err != nil {
		return nil, fmt.Errorf("loosey: getting latest version: %w", err)
	}
	if latest == 0 {
		return nil, ErrNoNextMigration
	}

	mig := m.findMigration(latest)
	if mig == nil {
		return nil, fmt.Errorf("%w: %d", ErrVersionNotFound, latest)
	}
	if len(mig.down) == 0 {
		return nil, fmt.Errorf("%w: version %d (%s)", ErrNoDownMigration, mig.version, mig.source)
	}

	var results []MigrationResult

	downResult := m.executeMigration(ctx, mig, DirectionDown)
	results = append(results, downResult)
	if downResult.Err != nil {
		return results, &MigrationError{Results: results}
	}

	upResult := m.executeMigration(ctx, mig, DirectionUp)
	results = append(results, upResult)
	if upResult.Err != nil {
		return results, &MigrationError{Results: results}
	}

	return results, nil
}

// Reset rolls back all applied migrations.
func (m *Manager[Q]) Reset(ctx context.Context) ([]MigrationResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	applied, err := m.q.ListApplied(ctx)
	if err != nil {
		return nil, fmt.Errorf("loosey: listing applied versions: %w", err)
	}
	appliedSet := make(map[int64]bool, len(applied))
	for _, v := range applied {
		appliedSet[v] = true
	}

	var results []MigrationResult
	for i := len(m.migrations) - 1; i >= 0; i-- {
		mig := m.migrations[i]
		if !appliedSet[mig.version] {
			continue
		}
		if len(mig.down) == 0 {
			return results, fmt.Errorf("%w: version %d (%s)", ErrNoDownMigration, mig.version, mig.source)
		}
		result := m.executeMigration(ctx, mig, DirectionDown)
		results = append(results, result)
		if result.Err != nil {
			return results, &MigrationError{Results: results}
		}
	}
	return results, nil
}

// Status returns the status of all known migrations.
func (m *Manager[Q]) Status(ctx context.Context) ([]MigrationStatus, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	applied, err := m.q.ListApplied(ctx)
	if err != nil {
		return nil, fmt.Errorf("loosey: listing applied versions: %w", err)
	}
	appliedSet := make(map[int64]bool, len(applied))
	for _, v := range applied {
		appliedSet[v] = true
	}

	var statuses []MigrationStatus
	for _, mig := range m.migrations {
		statuses = append(statuses, MigrationStatus{
			Version: mig.version,
			Name:    mig.name,
			Source:  mig.source,
			Applied: appliedSet[mig.version],
		})
	}
	return statuses, nil
}

// Version returns the highest applied migration version.
func (m *Manager[Q]) Version(ctx context.Context) (int64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.q.LatestVersion(ctx)
}

func (m *Manager[Q]) findMigration(version int64) *migration {
	for _, mig := range m.migrations {
		if mig.version == version {
			return mig
		}
	}
	return nil
}

func (m *Manager[Q]) executeMigration(ctx context.Context, mig *migration, dir Direction) MigrationResult {
	start := time.Now()
	result := MigrationResult{
		Version:   mig.version,
		Source:    mig.source,
		Direction: dir,
	}

	m.logger.LogAttrs(ctx, slog.LevelInfo, "applying migration", slog.Int64("version", mig.version), slog.String("source", mig.source), slog.String("direction", dir.String()))

	stmts := mig.up
	if dir == DirectionDown {
		stmts = mig.down
	}

	if mig.useTx {
		tx, err := m.db.BeginTx(ctx, nil)
		if err != nil {
			result.Duration = time.Since(start)
			result.Err = fmt.Errorf("loosey: begin transaction: %w", err)
			return result
		}

		for i, stmt := range stmts {
			m.logger.LogAttrs(ctx, slog.LevelDebug, "executing statement", slog.Int64("version", mig.version), slog.Int("index", i+1), slog.Int("of", len(stmts)))
			if _, err := tx.ExecContext(ctx, stmt); err != nil {
				_ = tx.Rollback()
				result.Duration = time.Since(start)
				result.Err = fmt.Errorf("statement %d of %d: %w", i+1, len(stmts), err)
				return result
			}
		}

		// Use tx-bound Queries for version tracking within the same transaction
		txQ := m.withTx(m.q, tx)
		if dir == DirectionUp {
			if err := txQ.InsertVersion(ctx, mig.version); err != nil {
				_ = tx.Rollback()
				result.Duration = time.Since(start)
				result.Err = fmt.Errorf("loosey: recording version: %w", err)
				return result
			}
		} else {
			if err := txQ.DeleteVersion(ctx, mig.version); err != nil {
				_ = tx.Rollback()
				result.Duration = time.Since(start)
				result.Err = fmt.Errorf("loosey: recording rollback: %w", err)
				return result
			}
		}

		if err := tx.Commit(); err != nil {
			result.Duration = time.Since(start)
			result.Err = fmt.Errorf("loosey: commit transaction: %w", err)
			return result
		}
	} else {
		for i, stmt := range stmts {
			m.logger.LogAttrs(ctx, slog.LevelDebug, "executing statement", slog.Int64("version", mig.version), slog.Int("index", i+1), slog.Int("of", len(stmts)))
			if _, err := m.db.ExecContext(ctx, stmt); err != nil {
				result.Duration = time.Since(start)
				result.Err = fmt.Errorf("statement %d of %d: %w", i+1, len(stmts), err)
				return result
			}
		}

		if dir == DirectionUp {
			if err := m.q.InsertVersion(ctx, mig.version); err != nil {
				result.Duration = time.Since(start)
				result.Err = fmt.Errorf("loosey: recording version: %w", err)
				return result
			}
		} else {
			if err := m.q.DeleteVersion(ctx, mig.version); err != nil {
				result.Duration = time.Since(start)
				result.Err = fmt.Errorf("loosey: recording rollback: %w", err)
				return result
			}
		}
	}

	result.Duration = time.Since(start)
	m.logger.LogAttrs(ctx, slog.LevelInfo, "applied migration", slog.Int64("version", mig.version), slog.String("source", mig.source), slog.String("direction", dir.String()), slog.Duration("duration", result.Duration))
	return result
}

package loosey

import (
	"errors"
	"fmt"
	"time"
)

type Direction int

const (
	DirectionUp Direction = iota
	DirectionDown
)

func (d Direction) String() string {
	if d == DirectionUp {
		return "up"
	}
	return "down"
}

type MigrationResult struct {
	Version   int64
	Source    string
	Direction Direction
	Duration  time.Duration
	Err       error
}

type ParseError struct {
	Source string
	Line   int
	Msg    string
}

func (e *ParseError) Error() string {
	if e.Line > 0 {
		return fmt.Sprintf("loosey: parse error in %s:%d: %s", e.Source, e.Line, e.Msg)
	}
	return fmt.Sprintf("loosey: parse error in %s: %s", e.Source, e.Msg)
}

type MigrationError struct {
	Results []MigrationResult
}

func (e *MigrationError) Error() string {
	for i := len(e.Results) - 1; i >= 0; i-- {
		if e.Results[i].Err != nil {
			return fmt.Sprintf("loosey: migration %d (%s) failed: %v", e.Results[i].Version, e.Results[i].Source, e.Results[i].Err)
		}
	}
	return "loosey: migration failed"
}

func (e *MigrationError) Unwrap() error {
	for i := len(e.Results) - 1; i >= 0; i-- {
		if e.Results[i].Err != nil {
			return e.Results[i].Err
		}
	}
	return nil
}

var (
	ErrNoMigrations    = errors.New("loosey: no migrations found")
	ErrNoNextMigration = errors.New("loosey: no next migration to apply")
	ErrAlreadyCurrent  = errors.New("loosey: database already at latest version")
	ErrVersionNotFound = errors.New("loosey: version not found")
	ErrOutOfOrder      = errors.New("loosey: out-of-order migration detected")
	ErrNoDownMigration = errors.New("loosey: migration has no down section")
)

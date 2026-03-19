package loosey

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

type parsedSQL struct {
	up    []string
	down  []string
	useTx bool
}

type parserState int

const (
	stateStart parserState = iota
	stateUp
	stateUpStatementBegin
	stateDown
	stateDownStatementBegin
)

type annotation int

const (
	annotationUp annotation = iota
	annotationDown
	annotationStatementBegin
	annotationStatementEnd
	annotationNoTransaction
	annotationEnvsubOn
	annotationEnvsubOff
)

func parseAnnotation(line string) (annotation, bool) {
	if len(line) == 0 || line[0] == ' ' || line[0] == '\t' {
		return 0, false
	}
	trimmed := strings.TrimSpace(line)
	if !strings.HasPrefix(trimmed, "--") {
		return 0, false
	}
	trimmed = strings.TrimSpace(trimmed[2:])
	if !strings.HasPrefix(strings.ToLower(trimmed), "+goose") {
		return 0, false
	}
	trimmed = strings.TrimSpace(trimmed[6:])
	upper := strings.ToUpper(trimmed)
	switch upper {
	case "UP":
		return annotationUp, true
	case "DOWN":
		return annotationDown, true
	case "STATEMENTBEGIN":
		return annotationStatementBegin, true
	case "STATEMENTEND":
		return annotationStatementEnd, true
	case "NO TRANSACTION":
		return annotationNoTransaction, true
	case "ENVSUB ON":
		return annotationEnvsubOn, true
	case "ENVSUB OFF":
		return annotationEnvsubOff, true
	}
	return 0, false
}

func endsWithSemicolon(line string) bool {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return false
	}
	// strip trailing single-line comment
	if idx := strings.Index(trimmed, "--"); idx >= 0 {
		trimmed = strings.TrimSpace(trimmed[:idx])
	}
	return len(trimmed) > 0 && trimmed[len(trimmed)-1] == ';'
}

func parseSQL(r io.Reader, env []string) (*parsedSQL, error) {
	result := &parsedSQL{useTx: true}
	state := stateStart
	envSubOn := false
	var buf strings.Builder
	scanner := bufio.NewScanner(r)

	for scanner.Scan() {
		line := scanner.Text()

		if ann, ok := parseAnnotation(line); ok {
			switch ann {
			case annotationUp:
				state = stateUp
				continue
			case annotationDown:
				// flush any accumulated up statement
				if buf.Len() > 0 {
					result.up = append(result.up, strings.TrimSpace(buf.String()))
					buf.Reset()
				}
				state = stateDown
				continue
			case annotationStatementBegin:
				if state == stateUp {
					state = stateUpStatementBegin
				} else if state == stateDown {
					state = stateDownStatementBegin
				}
				continue
			case annotationStatementEnd:
				stmt := strings.TrimSpace(buf.String())
				if stmt != "" {
					if state == stateUpStatementBegin {
						result.up = append(result.up, stmt)
					} else if state == stateDownStatementBegin {
						result.down = append(result.down, stmt)
					}
				}
				buf.Reset()
				if state == stateUpStatementBegin {
					state = stateUp
				} else if state == stateDownStatementBegin {
					state = stateDown
				}
				continue
			case annotationNoTransaction:
				result.useTx = false
				continue
			case annotationEnvsubOn:
				envSubOn = true
				continue
			case annotationEnvsubOff:
				envSubOn = false
				continue
			}
		}

		if envSubOn && state != stateStart {
			expanded, err := expandEnv(line, env)
			if err != nil {
				return nil, err
			}
			line = expanded
		}

		switch state {
		case stateStart:
			continue
		case stateUp:
			if buf.Len() > 0 {
				buf.WriteByte('\n')
			}
			buf.WriteString(line)
			if endsWithSemicolon(line) {
				result.up = append(result.up, strings.TrimSpace(buf.String()))
				buf.Reset()
			}
		case stateUpStatementBegin:
			if buf.Len() > 0 {
				buf.WriteByte('\n')
			}
			buf.WriteString(line)
		case stateDown:
			if buf.Len() > 0 {
				buf.WriteByte('\n')
			}
			buf.WriteString(line)
			if endsWithSemicolon(line) {
				result.down = append(result.down, strings.TrimSpace(buf.String()))
				buf.Reset()
			}
		case stateDownStatementBegin:
			if buf.Len() > 0 {
				buf.WriteByte('\n')
			}
			buf.WriteString(line)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("loosey: reading migration: %w", err)
	}

	// flush remaining buffer
	if buf.Len() > 0 {
		stmt := strings.TrimSpace(buf.String())
		if stmt != "" {
			switch state {
			case stateUp:
				result.up = append(result.up, stmt)
			case stateDown:
				result.down = append(result.down, stmt)
			}
		}
	}

	if state == stateStart {
		return nil, fmt.Errorf("loosey: no -- +goose Up annotation found")
	}

	return result, nil
}

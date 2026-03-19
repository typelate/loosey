package loosey

import (
	"reflect"
	"strings"
	"testing"
)

func TestParseSQL(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		env     []string
		want    parsedSQL
		wantErr bool
	}{
		{
			name:  "up only",
			input: "-- +goose Up\nCREATE TABLE t (id INT);\n",
			want:  parsedSQL{up: []string{"CREATE TABLE t (id INT);"}, useTx: true},
		},
		{
			name:  "up and down",
			input: "-- +goose Up\nCREATE TABLE t (id INT);\n\n-- +goose Down\nDROP TABLE t;\n",
			want:  parsedSQL{up: []string{"CREATE TABLE t (id INT);"}, down: []string{"DROP TABLE t;"}, useTx: true},
		},
		{
			name:  "multiple up statements",
			input: "-- +goose Up\nCREATE TABLE a (id INT);\nCREATE TABLE b (id INT);\n",
			want:  parsedSQL{up: []string{"CREATE TABLE a (id INT);", "CREATE TABLE b (id INT);"}, useTx: true},
		},
		{
			name:  "multiple down statements",
			input: "-- +goose Up\nCREATE TABLE a (id INT);\n-- +goose Down\nDROP TABLE b;\nDROP TABLE a;\n",
			want:  parsedSQL{up: []string{"CREATE TABLE a (id INT);"}, down: []string{"DROP TABLE b;", "DROP TABLE a;"}, useTx: true},
		},
		{
			name:  "case insensitive",
			input: "-- +GOOSE UP\nCREATE TABLE t (id INT);\n-- +Goose Down\nDROP TABLE t;\n",
			want:  parsedSQL{up: []string{"CREATE TABLE t (id INT);"}, down: []string{"DROP TABLE t;"}, useTx: true},
		},
		{
			name:  "statement begin end in up",
			input: "-- +goose Up\n-- +goose StatementBegin\nCREATE FUNCTION f() AS $$\nBEGIN\n  NULL;\nEND;\n$$;\n-- +goose StatementEnd\n",
			want:  parsedSQL{up: []string{"CREATE FUNCTION f() AS $$\nBEGIN\n  NULL;\nEND;\n$$;"}, useTx: true},
		},
		{
			name:  "statement begin end in down",
			input: "-- +goose Up\nSELECT 1;\n-- +goose Down\n-- +goose StatementBegin\nDROP FUNCTION f();\nDROP TABLE t;\n-- +goose StatementEnd\n",
			want:  parsedSQL{up: []string{"SELECT 1;"}, down: []string{"DROP FUNCTION f();\nDROP TABLE t;"}, useTx: true},
		},
		{
			name:  "no transaction",
			input: "-- +goose NO TRANSACTION\n-- +goose Up\nCREATE INDEX CONCURRENTLY idx ON t (id);\n",
			want:  parsedSQL{up: []string{"CREATE INDEX CONCURRENTLY idx ON t (id);"}, useTx: false},
		},
		{
			name:    "indented annotation ignored",
			input:   "  -- +goose Up\nCREATE TABLE t (id INT);\n",
			wantErr: true,
		},
		{
			name:    "tab indented annotation ignored",
			input:   "\t-- +goose Up\nCREATE TABLE t (id INT);\n",
			wantErr: true,
		},
		{
			name:    "missing up annotation",
			input:   "CREATE TABLE t (id INT);\n",
			wantErr: true,
		},
		{
			name:    "empty input",
			input:   "",
			wantErr: true,
		},
		{
			name:    "only whitespace",
			input:   "\n\n  \n",
			wantErr: true,
		},
		{
			name:  "empty up section",
			input: "-- +goose Up\n-- +goose Down\nDROP TABLE t;\n",
			want:  parsedSQL{down: []string{"DROP TABLE t;"}, useTx: true},
		},
		{
			name:  "empty down section",
			input: "-- +goose Up\nCREATE TABLE t (id INT);\n-- +goose Down\n",
			want:  parsedSQL{up: []string{"CREATE TABLE t (id INT);"}, useTx: true},
		},
		{
			name:  "envsub on",
			input: "-- +goose Up\n-- +goose ENVSUB ON\nCREATE TABLE ${TBL} (id INT);\n",
			env:   []string{"TBL=users"},
			want:  parsedSQL{up: []string{"CREATE TABLE users (id INT);"}, useTx: true},
		},
		{
			name:  "envsub toggle off",
			input: "-- +goose Up\n-- +goose ENVSUB ON\nCREATE TABLE ${TBL} (id INT);\n-- +goose ENVSUB OFF\nCREATE TABLE ${RAW} (id INT);\n",
			env:   []string{"TBL=users"},
			want:  parsedSQL{up: []string{"CREATE TABLE users (id INT);", "CREATE TABLE ${RAW} (id INT);"}, useTx: true},
		},
		{
			name:  "semicolon with trailing comment",
			input: "-- +goose Up\nINSERT INTO t VALUES (1); -- seed\n",
			want:  parsedSQL{up: []string{"INSERT INTO t VALUES (1); -- seed"}, useTx: true},
		},
		{
			name:  "up flush buffer at EOF without semicolon",
			input: "-- +goose Up\nCREATE TABLE t (id INT)",
			want:  parsedSQL{up: []string{"CREATE TABLE t (id INT)"}, useTx: true},
		},
		{
			name:  "down flush buffer at EOF without semicolon",
			input: "-- +goose Up\nSELECT 1;\n-- +goose Down\nDROP TABLE t",
			want:  parsedSQL{up: []string{"SELECT 1;"}, down: []string{"DROP TABLE t"}, useTx: true},
		},
		{
			name:  "accumulated up buffer flushes at down annotation",
			input: "-- +goose Up\nCREATE TABLE t (\n  id INT\n)\n-- +goose Down\nDROP TABLE t;\n",
			want:  parsedSQL{up: []string{"CREATE TABLE t (\n  id INT\n)"}, down: []string{"DROP TABLE t;"}, useTx: true},
		},
		{
			name:  "non-annotation comment preserved in SQL",
			input: "-- +goose Up\n-- regular comment\nCREATE TABLE t (id INT);\n",
			want:  parsedSQL{up: []string{"-- regular comment\nCREATE TABLE t (id INT);"}, useTx: true},
		},
		{
			name:  "lines before up annotation ignored",
			input: "-- this is a header\n-- +goose Up\nCREATE TABLE t (id INT);\n",
			want:  parsedSQL{up: []string{"CREATE TABLE t (id INT);"}, useTx: true},
		},
		{
			name:  "multiline down statement without semicolon",
			input: "-- +goose Up\nSELECT 1;\n-- +goose Down\nDROP TABLE\n  IF EXISTS t;\n",
			want:  parsedSQL{up: []string{"SELECT 1;"}, down: []string{"DROP TABLE\n  IF EXISTS t;"}, useTx: true},
		},
		{
			name:    "envsub error on undefined variable",
			input:   "-- +goose Up\n-- +goose ENVSUB ON\nCREATE TABLE ${MISSING} (id INT);\n",
			env:     []string{},
			wantErr: true,
		},
		{
			name:  "unknown annotation ignored",
			input: "-- +goose Up\n-- +goose FOOBAR\nCREATE TABLE t (id INT);\n",
			want:  parsedSQL{up: []string{"-- +goose FOOBAR\nCREATE TABLE t (id INT);"}, useTx: true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseSQL(strings.NewReader(tt.input), tt.env)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !reflect.DeepEqual(*got, tt.want) {
				t.Errorf("parseSQL result mismatch:\n  got:  %+v\n  want: %+v", *got, tt.want)
			}
		})
	}
}

func TestEndsWithSemicolon(t *testing.T) {
	tests := []struct {
		line string
		want bool
	}{
		{"CREATE TABLE t;", true},
		{"CREATE TABLE t; -- comment", true},
		{"CREATE TABLE t", false},
		{"", false},
		{"  ", false},
		{"-- just a comment", false},
	}
	for _, tt := range tests {
		if got := endsWithSemicolon(tt.line); got != tt.want {
			t.Errorf("endsWithSemicolon(%q) = %v, want %v", tt.line, got, tt.want)
		}
	}
}

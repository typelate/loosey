package loosey

import (
	"testing"
)

func TestExpandEnv_Simple(t *testing.T) {
	got, err := expandEnv("hello ${NAME}", []string{"NAME=world"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "hello world" {
		t.Errorf("got %q, want %q", got, "hello world")
	}
}

func TestExpandEnv_Default(t *testing.T) {
	got, err := expandEnv("${MISSING:-fallback}", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "fallback" {
		t.Errorf("got %q, want %q", got, "fallback")
	}
}

func TestExpandEnv_UndefinedError(t *testing.T) {
	_, err := expandEnv("${MISSING}", nil)
	if err == nil {
		t.Fatal("expected error for undefined var without default, got nil")
	}
}

func TestExpandEnv_Unclosed(t *testing.T) {
	_, err := expandEnv("${OOPS", nil)
	if err == nil {
		t.Fatal("expected error for unclosed ${, got nil")
	}
}

func TestExpandEnv_DollarEscape(t *testing.T) {
	got, err := expandEnv("cost is $$5", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "cost is $5" {
		t.Errorf("got %q, want %q", got, "cost is $5")
	}
}

func TestExpandEnv_BareDollar(t *testing.T) {
	got, err := expandEnv("price is $5", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "price is $5" {
		t.Errorf("got %q, want %q", got, "price is $5")
	}
}

func TestExpandEnv_EqualsInValue(t *testing.T) {
	got, err := expandEnv("${DSN}", []string{"DSN=host=localhost dbname=test"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "host=localhost dbname=test" {
		t.Errorf("got %q, want %q", got, "host=localhost dbname=test")
	}
}

package wp_test

import (
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestSlashUnslashString(t *testing.T) {
	in := `O'Reilly "quoted" \path`
	slashed := wp.SlashString(in)
	if slashed == in {
		t.Fatalf("expected slashes added, got %q", slashed)
	}
	out := wp.UnslashString(slashed)
	if out != in {
		t.Fatalf("roundtrip: got %q want %q (slashed=%q)", out, in, slashed)
	}
}

func TestSlashUnslashNested(t *testing.T) {
	in := []any{"a'b", map[string]any{"k": `c"d`}, 42}
	slashed := wp.Slash(in).([]any)
	if slashed[0].(string) == "a'b" {
		t.Fatal("expected first element slashed")
	}
	if slashed[2].(int) != 42 {
		t.Fatalf("non-string should pass through: %v", slashed[2])
	}
	out := wp.Unslash(slashed).([]any)
	if out[0].(string) != "a'b" {
		t.Fatalf("unslash[0]=%q", out[0])
	}
	m := out[1].(map[string]any)
	if m["k"].(string) != `c"d` {
		t.Fatalf("unslash map: %#v", m)
	}
}

func TestSlashStringSlice(t *testing.T) {
	in := []string{`x'y`, `a\b`}
	slashed := wp.Slash(in).([]string)
	out := wp.Unslash(slashed).([]string)
	if out[0] != in[0] || out[1] != in[1] {
		t.Fatalf("roundtrip %#v -> %#v", in, out)
	}
}

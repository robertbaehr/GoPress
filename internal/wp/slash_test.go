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

func TestSlashMapStringString(t *testing.T) {
	in := map[string]string{"a": `O'Reilly`, "b": `c"d`}
	slashed := wp.Slash(in).(map[string]string)
	if slashed["a"] == in["a"] {
		t.Fatal("expected map values slashed")
	}
	out := wp.Unslash(slashed).(map[string]string)
	if out["a"] != in["a"] || out["b"] != in["b"] {
		t.Fatalf("roundtrip %#v -> %#v", in, out)
	}
}

func TestSlashMapAnyAny(t *testing.T) {
	in := map[any]any{1: `a'b`, "k": []any{`c"d`, 7}}
	slashed := wp.Slash(in).(map[any]any)
	if slashed[1].(string) == `a'b` {
		t.Fatal("expected nested slash")
	}
	out := wp.Unslash(slashed).(map[any]any)
	if out[1].(string) != `a'b` {
		t.Fatalf("unslash[1]=%q", out[1])
	}
	inner := out["k"].([]any)
	if inner[0].(string) != `c"d` || inner[1].(int) != 7 {
		t.Fatalf("nested %#v", inner)
	}
}

func TestStripslashesDeepAlias(t *testing.T) {
	in := `she\'s`
	if wp.StripslashesDeep(in).(string) != wp.UnslashString(in) {
		t.Fatal("StripslashesDeep should match Unslash")
	}
	if wp.StripslashesFromStringsOnly(42).(int) != 42 {
		t.Fatal("non-string should pass through")
	}
	if wp.StripslashesFromStringsOnly(`a\'b`).(string) != "a'b" {
		t.Fatal("expected string strip")
	}
}

// P2.124: map[string][]string / []map[string]any / MapDeep.
func TestUnslashMapStringStringSlice(t *testing.T) {
	in := map[string][]string{"q": {`a\'b`, `c\"d`}}
	out := wp.Unslash(in).(map[string][]string)
	if out["q"][0] != "a'b" || out["q"][1] != `c"d` {
		t.Fatalf("got %#v", out)
	}
}

func TestUnslashSliceOfMaps(t *testing.T) {
	in := []map[string]any{{"k": `x\'y`}, {"n": 7}}
	out := wp.Unslash(in).([]map[string]any)
	if out[0]["k"].(string) != "x'y" {
		t.Fatalf("got %#v", out[0])
	}
	if out[1]["n"].(int) != 7 {
		t.Fatalf("non-string leaf: %#v", out[1])
	}
}

func TestMapDeepIdentity(t *testing.T) {
	in := map[string][]any{"a": {"b", 1}}
	out := wp.MapDeep(in, func(v any) any { return v }).(map[string][]any)
	if out["a"][0].(string) != "b" || out["a"][1].(int) != 1 {
		t.Fatalf("got %#v", out)
	}
}

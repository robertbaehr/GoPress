package wp_test

import (
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// P2.138: quote-aware script/style open tags + deep strip.
func TestWPStripAllTagsQuotedScriptStyleP2138(t *testing.T) {
	got := wp.WPStripAllTags(`before<script data-x="a>b">evil()</script>after`, false)
	if got != "beforeafter" {
		t.Fatalf("quoted script attr: got %q", got)
	}
	got = wp.WPStripAllTags(`x<style title='y>z'>.a{}</style>y`, false)
	if got != "xy" {
		t.Fatalf("quoted style attr: got %q", got)
	}
	// Non-tag prefix must not be eaten.
	got = wp.WPStripAllTags(`<scripted>keep</scripted>`, false)
	if got != "keep" {
		t.Fatalf("scripted false-positive: got %q", got)
	}
}

func TestWPStripAllTagsDeepP2138(t *testing.T) {
	in := map[string]any{
		"a": `<b>bold</b>`,
		"b": []any{`<script>x</script>ok`, map[string]any{"c": `<i>1</i>`}},
	}
	out := wp.WPStripAllTagsDeep(in).(map[string]any)
	if out["a"] != "bold" {
		t.Fatalf("a=%v", out["a"])
	}
	arr := out["b"].([]any)
	if arr[0] != "ok" {
		t.Fatalf("b[0]=%v", arr[0])
	}
	inner := arr[1].(map[string]any)
	if inner["c"] != "1" {
		t.Fatalf("b[1].c=%v", inner["c"])
	}
}

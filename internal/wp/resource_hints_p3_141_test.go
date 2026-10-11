package wp_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// P3.141: modulepreload http(s)-only, case-insensitive href dedupe, as/type/id attrs.
func TestResourceHintsModulePreloadP3141(t *testing.T) {
	c := &wp.Context{Hooks: wp.NewHooks()}
	c.Hooks.AddFilter("wp_resource_hints", func(value any, args ...any) any {
		rel, _ := args[0].(string)
		if rel != "modulepreload" {
			return value
		}
		return []any{
			"https://CDN.example.com/block-editor.js",
			"https://cdn.example.com/block-editor.js",
			map[string]any{
				"href":         "https://cdn.example.com/interactivity.js",
				"as":           "script",
				"type":         "text/javascript",
				"id":           "interactivity-js-modulepreload",
				"crossorigin":  "anonymous",
			},
			"ftp://files.example.com/mod.js",
			"//protocol.relative.example/mod.js",
		}
	}, 10, 2)

	got := wp.ResourceHintsHTML(c)
	if strings.Count(strings.ToLower(got), "cdn.example.com/block-editor.js") != 1 {
		t.Fatalf("expected single block-editor modulepreload, got %q", got)
	}
	if !strings.Contains(got, `rel='modulepreload'`) {
		t.Fatalf("missing modulepreload rel: %q", got)
	}
	for _, want := range []string{
		`as='script'`,
		`type='text/javascript'`,
		`id='interactivity-js-modulepreload'`,
		`crossorigin='anonymous'`,
		`href='https://cdn.example.com/interactivity.js'`,
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %q in %q", want, got)
		}
	}
	if strings.Contains(got, "ftp://") || strings.Contains(got, "protocol.relative") {
		t.Fatalf("non-http(s) should be rejected: %q", got)
	}
	// Default modulepreload empty for corpus parity.
	if empty := wp.ResourceHintsHTML(&wp.Context{Hooks: wp.NewHooks()}); strings.Contains(empty, "modulepreload") {
		t.Fatalf("default must stay empty: %q", empty)
	}
}

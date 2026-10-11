package wp_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// P3.140: prefetch http(s)-only, case-insensitive href dedupe, as/type/pr attrs.
func TestResourceHintsPrefetchDedupAttrsP3140(t *testing.T) {
	c := &wp.Context{Hooks: wp.NewHooks()}
	c.Hooks.AddFilter("wp_resource_hints", func(value any, args ...any) any {
		rel, _ := args[0].(string)
		if rel != "prefetch" {
			return value
		}
		return []any{
			"https://CDN.example.com/app.js",
			"https://cdn.example.com/app.js",
			map[string]any{
				"href": "https://cdn.example.com/style.css",
				"as":   "style",
				"type": "text/css",
				"pr":   0.75,
			},
			"ftp://files.example.com/x",
			"//protocol.relative.example/x",
			"mailto:nobody@example.com",
		}
	}, 10, 2)

	got := wp.ResourceHintsHTML(c)
	if strings.Count(strings.ToLower(got), "cdn.example.com/app.js") != 1 {
		t.Fatalf("expected single app.js prefetch, got %q", got)
	}
	if !strings.Contains(got, `rel='prefetch'`) {
		t.Fatalf("missing prefetch rel: %q", got)
	}
	for _, want := range []string{
		`as='style'`,
		`type='text/css'`,
		`pr='0.75'`,
		`href='https://cdn.example.com/style.css'`,
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %q in %q", want, got)
		}
	}
	if strings.Contains(got, "ftp://") || strings.Contains(got, "mailto:") || strings.Contains(got, "protocol.relative") {
		t.Fatalf("non-http(s) should be rejected: %q", got)
	}
	// Default prefetch empty for corpus parity.
	if empty := wp.ResourceHintsHTML(&wp.Context{Hooks: wp.NewHooks()}); strings.Contains(empty, "prefetch") {
		t.Fatalf("default must stay empty: %q", empty)
	}
}

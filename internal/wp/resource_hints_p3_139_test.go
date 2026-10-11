package wp_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// P3.139: preconnect dedupes scheme+host case-insensitively; keeps crossorigin.
func TestResourceHintsPreconnectDedupP3139(t *testing.T) {
	c := &wp.Context{Hooks: wp.NewHooks()}
	c.Hooks.AddFilter("wp_resource_hints", func(value any, args ...any) any {
		rel, _ := args[0].(string)
		if rel != "preconnect" {
			return value
		}
		return []any{
			"https://FONTS.gstatic.com/css",
			"https://fonts.gstatic.com/extra",
			map[string]any{"href": "https://cdn.example.com", "crossorigin": "anonymous"},
			"fonts.example.com",
			"//fonts.example.com/path",
		}
	}, 10, 2)

	got := wp.ResourceHintsHTML(c)
	lower := strings.ToLower(got)
	if strings.Count(lower, "fonts.gstatic.com") != 1 {
		t.Fatalf("expected single fonts.gstatic.com preconnect, got %q", got)
	}
	if !strings.Contains(got, `rel='preconnect'`) || !strings.Contains(lower, `href='https://fonts.gstatic.com'`) {
		t.Fatalf("preconnect href: %q", got)
	}
	if !strings.Contains(got, `crossorigin='anonymous'`) {
		t.Fatalf("expected crossorigin preserved: %q", got)
	}
	// Bare host → protocol-relative; dedupe with //fonts.example.com.
	if strings.Count(strings.ToLower(got), "fonts.example.com") != 1 {
		t.Fatalf("expected single fonts.example.com, got %q", got)
	}
	// Default preconnect empty.
	if empty := wp.ResourceHintsHTML(&wp.Context{Hooks: wp.NewHooks()}); strings.Contains(empty, "preconnect") {
		t.Fatalf("default must stay empty: %q", empty)
	}
}

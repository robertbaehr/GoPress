package wp_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// P3.137: default resource hints empty; filter urls emit dns-prefetch/preconnect.
func TestResourceHintsDefaultEmptyP3137(t *testing.T) {
	c := &wp.Context{Hooks: wp.NewHooks()}
	if got := wp.ResourceHintsHTML(c); got != "" {
		t.Fatalf("default want empty, got %q", got)
	}
	if got := wp.ResourceHintsHTML(nil); got != "" {
		t.Fatalf("nil want empty, got %q", got)
	}
}

func TestResourceHintsFilterP3137(t *testing.T) {
	c := &wp.Context{Hooks: wp.NewHooks()}
	c.Hooks.AddFilter("wp_resource_hints", func(value any, args ...any) any {
		rel, _ := args[0].(string)
		urls, _ := value.([]any)
		if rel == "dns-prefetch" {
			return append(urls, "https://fonts.googleapis.com")
		}
		if rel == "preconnect" {
			return append(urls, map[string]any{
				"href":        "https://example.com",
				"crossorigin": "anonymous",
			})
		}
		return urls
	}, 10, 2)

	got := wp.ResourceHintsHTML(c)
	if !strings.Contains(got, `rel='dns-prefetch'`) || !strings.Contains(got, `href='//fonts.googleapis.com'`) {
		t.Fatalf("dns-prefetch: %q", got)
	}
	if !strings.Contains(got, `rel='preconnect'`) || !strings.Contains(got, `href='https://example.com'`) {
		t.Fatalf("preconnect: %q", got)
	}
	if !strings.Contains(got, `crossorigin='anonymous'`) {
		t.Fatalf("crossorigin: %q", got)
	}
}

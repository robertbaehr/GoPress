package wp_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// P3.160: ContentURL builds siteurl/wp-content[+path] + content_url filter.
func TestContentURLP3160(t *testing.T) {
	c := testBootQuery(t)
	c.Options.Cache.Set("siteurl", "http://example.test", "options")
	got := wp.ContentURL(c, "")
	if !strings.HasSuffix(got, "/wp-content") {
		t.Fatalf("base=%q", got)
	}
	got = wp.ContentURL(c, "uploads/2024/x.png")
	if !strings.HasSuffix(got, "/wp-content/uploads/2024/x.png") {
		t.Fatalf("path=%q", got)
	}
	c.Hooks.AddFilter("content_url", func(value any, _ ...any) any {
		return "http://cdn.example/content"
	}, 10, 2)
	if got := wp.ContentURL(c, "x"); got != "http://cdn.example/content" {
		t.Fatalf("filter=%q", got)
	}
}

package wp_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// P2.163: IncludesURL builds siteurl/wp-includes/[path] + includes_url filter.
func TestIncludesURLP2163(t *testing.T) {
	c := testBootQuery(t)
	c.Options.Cache.Set("siteurl", "http://example.test", "options")
	got := wp.IncludesURL(c, "", "")
	if !strings.Contains(got, "/wp-includes/") {
		t.Fatalf("base=%q", got)
	}
	got = wp.IncludesURL(c, "js/wp-util.js", "")
	if !strings.HasSuffix(got, "/wp-includes/js/wp-util.js") {
		t.Fatalf("path=%q", got)
	}
	c.Hooks.AddFilter("includes_url", func(value any, _ ...any) any {
		return "http://cdn.example/includes/"
	}, 10, 3)
	if got := wp.IncludesURL(c, "x.js", ""); got != "http://cdn.example/includes/" {
		t.Fatalf("filter=%q", got)
	}
}

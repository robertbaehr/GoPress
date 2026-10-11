package wp_test

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestGetShortlinkPost(t *testing.T) {
	if os.Getenv("GOPRESS_SKIP_DB") != "" {
		t.Skip("GOPRESS_SKIP_DB set")
	}
	c, err := wp.Boot(context.Background(), testBootConfig())
	if err != nil {
		t.Skipf("Boot unavailable: %v", err)
	}
	defer c.DB.Close()

	got := wp.GetShortlink(c, 1, "post", 0)
	if !strings.Contains(got, "/?p=1") {
		t.Fatalf("shortlink=%q", got)
	}
	query := wp.GetShortlink(c, 0, "query", 1)
	if query != got {
		t.Fatalf("query context %q != post context %q", query, got)
	}
	if empty := wp.GetShortlink(c, 0, "query", 0); empty != "" {
		t.Fatalf("expected empty without id, got %q", empty)
	}
}

func TestGetShortlinkFilters(t *testing.T) {
	// P3.106: pre_get_shortlink short-circuit + get_shortlink filter.
	if os.Getenv("GOPRESS_SKIP_DB") != "" {
		t.Skip("GOPRESS_SKIP_DB set")
	}
	c, err := wp.Boot(context.Background(), testBootConfig())
	if err != nil {
		t.Skipf("Boot unavailable: %v", err)
	}
	defer c.DB.Close()

	c.Hooks.AddFilter("pre_get_shortlink", func(value any, args ...any) any {
		return "https://example.test/s/1"
	}, 10, 1)
	got := wp.GetShortlink(c, 1, "post", 0)
	if got != "https://example.test/s/1" {
		t.Fatalf("pre_get_shortlink: %q", got)
	}

	c2, err := wp.Boot(context.Background(), testBootConfig())
	if err != nil {
		t.Skipf("Boot unavailable: %v", err)
	}
	defer c2.DB.Close()
	c2.Hooks.AddFilter("get_shortlink", func(value any, args ...any) any {
		s, _ := value.(string)
		return s + "&utm=gopress"
	}, 10, 1)
	got = wp.GetShortlink(c2, 1, "post", 0)
	if !strings.Contains(got, "/?p=1") || !strings.HasSuffix(got, "&utm=gopress") {
		t.Fatalf("get_shortlink: %q", got)
	}
	allow := wp.GetShortlinkAllowSlugs(c2, 1, "post", 0, false)
	if allow != got {
		t.Fatalf("allow_slugs should not change core shortlink: %q vs %q", allow, got)
	}
}

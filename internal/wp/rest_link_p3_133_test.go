package wp_test

import (
	"context"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// P3.133: RestOutputLinkHeader emits api.w.org Link via GetRestURL.
func TestRestOutputLinkHeaderP3133(t *testing.T) {
	if os.Getenv("GOPRESS_SKIP_DB") != "" {
		t.Skip("GOPRESS_SKIP_DB set")
	}
	c, err := wp.Boot(context.Background(), testBootConfig())
	if err != nil {
		t.Skipf("Boot unavailable: %v", err)
	}
	defer c.DB.Close()

	rec := httptest.NewRecorder()
	wp.RestOutputLinkHeader(rec, c)
	got := rec.Header().Get("Link")
	if !strings.Contains(got, `/wp-json/`) || !strings.Contains(got, `rel="https://api.w.org/"`) {
		t.Fatalf("expected api.w.org Link, got %q", got)
	}

	// rest_url filter override.
	c2, err := wp.Boot(context.Background(), testBootConfig())
	if err != nil {
		t.Skipf("Boot unavailable: %v", err)
	}
	defer c2.DB.Close()
	c2.Hooks.AddFilter("rest_url", func(value any, args ...any) any {
		return "https://example.test/custom-json/"
	}, 10, 4)
	rec2 := httptest.NewRecorder()
	wp.RestOutputLinkHeader(rec2, c2)
	got2 := rec2.Header().Get("Link")
	if !strings.Contains(got2, "https://example.test/custom-json/") {
		t.Fatalf("filter rest_url header: %q", got2)
	}

	// GetRestURL path append.
	root := wp.GetRestURL(c, "/")
	if !strings.HasSuffix(root, "/wp-json/") {
		t.Fatalf("GetRestURL root: %q", root)
	}
	item := wp.RestURL(c, "/wp/v2/posts/1")
	if !strings.Contains(item, "/wp-json/wp/v2/posts/1") {
		t.Fatalf("RestURL item: %q", item)
	}
}

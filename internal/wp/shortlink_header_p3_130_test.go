package wp_test

import (
	"context"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// P3.130: ShortlinkHeader uses query context and emits rel=shortlink.
func TestShortlinkHeaderP3130(t *testing.T) {
	if os.Getenv("GOPRESS_SKIP_DB") != "" {
		t.Skip("GOPRESS_SKIP_DB set")
	}
	c, err := wp.Boot(context.Background(), testBootConfig())
	if err != nil {
		t.Skipf("Boot unavailable: %v", err)
	}
	defer c.DB.Close()

	rec := httptest.NewRecorder()
	wp.ShortlinkHeader(rec, c, 1)
	vals := rec.Header().Values("Link")
	found := false
	for _, v := range vals {
		if strings.Contains(v, "rel=shortlink") && strings.Contains(v, "?p=1") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected shortlink Link header, got %v", vals)
	}

	// Empty queried ID → no header.
	rec2 := httptest.NewRecorder()
	wp.ShortlinkHeader(rec2, c, 0)
	if len(rec2.Header().Values("Link")) != 0 {
		t.Fatalf("expected no shortlink for queriedID=0, got %v", rec2.Header().Values("Link"))
	}

	// pre_get_shortlink short-circuit with query context.
	c2, err := wp.Boot(context.Background(), testBootConfig())
	if err != nil {
		t.Skipf("Boot unavailable: %v", err)
	}
	defer c2.DB.Close()
	c2.Hooks.AddFilter("pre_get_shortlink", func(value any, args ...any) any {
		return "http://example.test/s"
	}, 10, 4)
	rec3 := httptest.NewRecorder()
	wp.ShortlinkHeader(rec3, c2, 1)
	got := rec3.Header().Get("Link")
	if !strings.Contains(got, "http://example.test/s") || !strings.Contains(got, "rel=shortlink") {
		t.Fatalf("filter shortlink header: %q", got)
	}
}

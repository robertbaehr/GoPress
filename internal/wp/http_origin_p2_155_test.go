package wp_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// P2.155: GetHTTPOrigin reads Origin header + http_origin filter.
func TestGetHTTPOriginP2155(t *testing.T) {
	if got := wp.GetHTTPOrigin(nil); got != "" {
		t.Fatalf("nil=%q", got)
	}
	c := testBootQuery(t)
	req := httptest.NewRequest(http.MethodGet, "http://example.test/", nil)
	req.Header.Set("Origin", "https://front.example")
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	if got := wp.GetHTTPOrigin(c); got != "https://front.example" {
		t.Fatalf("origin=%q", got)
	}
	c.Hooks.AddFilter("http_origin", func(value any, _ ...any) any {
		return "https://filtered.example"
	}, 10, 1)
	if got := wp.GetHTTPOrigin(c); got != "https://filtered.example" {
		t.Fatalf("filtered=%q", got)
	}
}

// P2.155: HttpSupports ssl always; unknown caps false; https URL implies ssl.
func TestHttpSupportsP2155(t *testing.T) {
	if !wp.HttpSupports(nil, "") {
		t.Fatal("empty caps")
	}
	if !wp.HttpSupports(map[string]bool{"ssl": true}, "") {
		t.Fatal("ssl")
	}
	if wp.HttpSupports(map[string]bool{"cookies": true}, "") {
		t.Fatal("unknown cookies")
	}
	if !wp.HttpSupports(nil, "https://example.com/") {
		t.Fatal("https implies ssl")
	}
	if !wp.HttpSupportsList([]string{"ssl"}, "http://example.com/") {
		t.Fatal("list ssl")
	}
	if wp.HttpSupportsList([]string{"ssl", "foo"}, "") {
		t.Fatal("list unknown")
	}
}

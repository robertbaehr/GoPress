package wp_test

import (
	"net/http/httptest"
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// P3.132: ClearPingbackHeaders splits comma-joined Link and spaced rel tokens.
func TestClearPingbackHeadersDeepP3132(t *testing.T) {
	rec := httptest.NewRecorder()
	rec.Header().Set("X-Pingback", "http://example.com/xmlrpc.php")
	rec.Header().Set("Link",
		`<http://example.com/xmlrpc.php>; rel = "pingback", <http://example.com/wp-json/>; rel="https://api.w.org/"`)
	wp.ClearPingbackHeaders(rec)
	if got := rec.Header().Get("X-Pingback"); got != "" {
		t.Fatalf("X-Pingback still set: %q", got)
	}
	links := rec.Header().Values("Link")
	if len(links) != 1 {
		t.Fatalf("Link values=%v", links)
	}
	if links[0] != `<http://example.com/wp-json/>; rel="https://api.w.org/"` {
		t.Fatalf("kept Link=%q", links[0])
	}
	if wp.IsPingbackLink(`<http://x/xmlrpc.php>; rel=pingback`) != true {
		t.Fatal("expected IsPingbackLink true")
	}
	if wp.IsPingbackLink(`<http://x/wp-json/>; rel="https://api.w.org/"`) {
		t.Fatal("expected IsPingbackLink false for api.w.org")
	}
}

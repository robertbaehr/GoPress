package wp_test

import (
	"net/http/httptest"
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// P3.131: ClearPingbackHeaders drops X-Pingback and Link rel=pingback.
func TestClearPingbackHeadersP3131(t *testing.T) {
	rec := httptest.NewRecorder()
	rec.Header().Set("X-Pingback", "http://example.com/xmlrpc.php")
	rec.Header().Add("Link", `<http://example.com/xmlrpc.php>; rel="pingback"`)
	rec.Header().Add("Link", `<http://example.com/wp-json/>; rel="https://api.w.org/"`)
	wp.ClearPingbackHeaders(rec)
	if got := rec.Header().Get("X-Pingback"); got != "" {
		t.Fatalf("X-Pingback still set: %q", got)
	}
	links := rec.Header().Values("Link")
	if len(links) != 1 || links[0] != `<http://example.com/wp-json/>; rel="https://api.w.org/"` {
		t.Fatalf("links=%v", links)
	}
}

// P3.131: nil ResponseWriter is a no-op.
func TestClearPingbackHeadersNilP3131(t *testing.T) {
	wp.ClearPingbackHeaders(nil)
}

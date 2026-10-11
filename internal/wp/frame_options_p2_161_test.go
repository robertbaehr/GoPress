package wp_test

import (
	"net/http/httptest"
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// P2.161: SendFrameOptionsHeader sets SAMEORIGIN + CSP frame-ancestors.
func TestSendFrameOptionsHeaderP2161(t *testing.T) {
	rec := httptest.NewRecorder()
	wp.SendFrameOptionsHeader(rec)
	if got := rec.Header().Get("X-Frame-Options"); got != "SAMEORIGIN" {
		t.Fatalf("X-Frame-Options=%q", got)
	}
	if got := rec.Header().Get("Content-Security-Policy"); got != "frame-ancestors 'self'" {
		t.Fatalf("CSP=%q", got)
	}
	wp.SendFrameOptionsHeader(nil) // no panic
}

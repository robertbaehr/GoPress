package wp_test

import (
	"net/http/httptest"
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// P2.159: SendNosniffHeader sets X-Content-Type-Options.
func TestSendNosniffHeaderP2159(t *testing.T) {
	rec := httptest.NewRecorder()
	wp.SendNosniffHeader(rec)
	if got := rec.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Fatalf("got=%q", got)
	}
	wp.SendNosniffHeader(nil) // no panic
}

// P2.159: AdminHeaders default + admin_referrer_policy filter.
func TestAdminHeadersP2159(t *testing.T) {
	c := testBootQuery(t)
	rec := httptest.NewRecorder()
	wp.AdminHeaders(rec, c)
	if got := rec.Header().Get("Referrer-Policy"); got != "strict-origin-when-cross-origin" {
		t.Fatalf("default=%q", got)
	}

	c.Hooks.AddFilter("admin_referrer_policy", func(value any, _ ...any) any {
		return "same-origin"
	}, 10, 1)
	rec2 := httptest.NewRecorder()
	wp.AdminHeaders(rec2, c)
	if got := rec2.Header().Get("Referrer-Policy"); got != "same-origin" {
		t.Fatalf("filtered=%q", got)
	}

	c.Hooks.AddFilter("admin_referrer_policy", func(value any, _ ...any) any {
		return "  "
	}, 20, 1)
	rec3 := httptest.NewRecorder()
	wp.AdminHeaders(rec3, c)
	if got := rec3.Header().Get("Referrer-Policy"); got != "" {
		t.Fatalf("empty policy should skip, got %q", got)
	}
	wp.AdminHeaders(nil, c)
}

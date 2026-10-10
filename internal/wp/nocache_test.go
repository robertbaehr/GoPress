package wp_test

import (
	"net/http/httptest"
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestNoCacheHeaders(t *testing.T) {
	rec := httptest.NewRecorder()
	wp.NoCacheHeaders(rec)
	if got := rec.Header().Get("Expires"); got != "Wed, 11 Jan 1984 05:00:00 GMT" {
		t.Fatalf("Expires=%q", got)
	}
	want := "no-cache, must-revalidate, max-age=0, no-store, private"
	if got := rec.Header().Get("Cache-Control"); got != want {
		t.Fatalf("Cache-Control=%q want %q", got, want)
	}
}

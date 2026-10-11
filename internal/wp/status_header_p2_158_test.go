package wp_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// P2.158: GetStatusHeaderDesc covers common codes; unknown empty.
func TestGetStatusHeaderDescP2158(t *testing.T) {
	if got := wp.GetStatusHeaderDesc(200); got != "OK" {
		t.Fatalf("200=%q", got)
	}
	if got := wp.GetStatusHeaderDesc(404); got != "Not Found" {
		t.Fatalf("404=%q", got)
	}
	if got := wp.GetStatusHeaderDesc(418); got != "I'm a teapot" {
		t.Fatalf("418=%q", got)
	}
	if got := wp.GetStatusHeaderDesc(999); got != "" {
		t.Fatalf("unknown=%q", got)
	}
}

// P2.158: StatusHeader writes status + filtered status line header.
func TestStatusHeaderP2158(t *testing.T) {
	c := testBootQuery(t)
	req := httptest.NewRequest(http.MethodGet, "http://example.test/", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	wp.StatusHeader(rec, c, 403, "")
	if rec.Code != 403 {
		t.Fatalf("code=%d", rec.Code)
	}
	line := rec.Header().Get("X-GoPress-Status-Line")
	if !strings.Contains(line, "403") || !strings.Contains(line, "Forbidden") {
		t.Fatalf("line=%q", line)
	}

	rec = httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	c.Hooks.AddFilter("status_header", func(value any, _ ...any) any {
		return "HTTP/1.1 418 Custom Teapot"
	}, 10, 4)
	wp.StatusHeader(rec, c, 418, "")
	if got := rec.Header().Get("X-GoPress-Status-Line"); got != "HTTP/1.1 418 Custom Teapot" {
		t.Fatalf("filtered=%q", got)
	}
}

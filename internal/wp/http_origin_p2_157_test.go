package wp_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// P2.157: SendOriginHeaders sets CORS headers for allowed Origin.
func TestSendOriginHeadersAllowedP2157(t *testing.T) {
	c := testBootQuery(t)
	home := strings.TrimRight(wp.HomeURL(c, "", ""), "/")
	u := strings.TrimPrefix(strings.TrimPrefix(home, "https://"), "http://")
	host := u
	if i := strings.IndexByte(u, '/'); i >= 0 {
		host = u[:i]
	}
	allowed := "http://" + host

	req := httptest.NewRequest(http.MethodGet, "http://example.test/", nil)
	req.Header.Set("Origin", allowed)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	got := wp.SendOriginHeaders(c)
	if got != allowed {
		t.Fatalf("got=%q want %q", got, allowed)
	}
	if rec.Header().Get("Access-Control-Allow-Origin") != allowed {
		t.Fatalf("ACAO=%q", rec.Header().Get("Access-Control-Allow-Origin"))
	}
	if rec.Header().Get("Access-Control-Allow-Credentials") != "true" {
		t.Fatalf("ACAC=%q", rec.Header().Get("Access-Control-Allow-Credentials"))
	}
}

// P2.157: denied OPTIONS → 403; non-OPTIONS denied → empty without status write.
func TestSendOriginHeadersDeniedP2157(t *testing.T) {
	c := testBootQuery(t)
	req := httptest.NewRequest(http.MethodOptions, "http://example.test/", nil)
	req.Header.Set("Origin", "https://evil.example")
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	if got := wp.SendOriginHeaders(c); got != "" {
		t.Fatalf("denied OPTIONS got=%q", got)
	}
	if rec.Code != 403 {
		t.Fatalf("status=%d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "http://example.test/", nil)
	req.Header.Set("Origin", "https://evil.example")
	rec = httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	if got := wp.SendOriginHeaders(c); got != "" {
		t.Fatalf("denied GET got=%q", got)
	}
	if rec.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Fatal("should not set ACAO")
	}
}

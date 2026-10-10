package server_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/server"
)

func TestTrackbackGone(t *testing.T) {
	s := &server.Server{}
	for _, method := range []string{http.MethodGet, http.MethodPost} {
		req := httptest.NewRequest(method, "/wp-trackback.php", strings.NewReader("title=x&url=http://example.com/&excerpt=y&blog_name=z"))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rec := httptest.NewRecorder()
		s.ServeHTTP(rec, req)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("%s status=%d body=%q", method, rec.Code, rec.Body.String())
		}
		ct := rec.Header().Get("Content-Type")
		if !strings.Contains(ct, "text/xml") {
			t.Fatalf("%s Content-Type=%q", method, ct)
		}
		body := rec.Body.String()
		if !strings.Contains(body, "<error>1</error>") || !strings.Contains(body, "trackbacks are closed") {
			t.Fatalf("%s body=%q", method, body)
		}
		if rec.Header().Get("X-Robots-Tag") != "noindex" {
			t.Fatalf("%s X-Robots-Tag=%q", method, rec.Header().Get("X-Robots-Tag"))
		}
		if got := rec.Header().Get("Allow"); got != "GET, POST" {
			t.Fatalf("%s Allow=%q", method, got)
		}
	}

	req := httptest.NewRequest(http.MethodHead, "/wp-trackback.php", nil)
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("HEAD status=%d", rec.Code)
	}
	if rec.Body.Len() != 0 {
		t.Fatalf("HEAD body should be empty: %q", rec.Body.String())
	}
	if !strings.Contains(rec.Header().Get("Content-Type"), "text/xml") {
		t.Fatalf("HEAD Content-Type=%q", rec.Header().Get("Content-Type"))
	}
}

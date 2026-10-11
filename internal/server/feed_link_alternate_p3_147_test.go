package server_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/server"
)

// P3.147: front head feed alternates use application/rss+xml + /feed/ hrefs.
func TestFrontFeedLinkAlternateP3147(t *testing.T) {
	c := testBoot(t)
	srv := &server.Server{WP: c, AbsPath: "/workspace/", Front: &server.Front{}}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d", rec.Code)
	}
	body := rec.Body.String()
	for _, want := range []string{
		`rel="alternate"`,
		`type="application/rss+xml"`,
		`/feed/`,
		`/comments/feed/`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing %q in head: %q", want, body[:min(1500, len(body))])
		}
	}
}

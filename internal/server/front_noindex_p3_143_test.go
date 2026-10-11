package server_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/server"
)

// P3.143: ?replytocom= adds noindex via wp_robots_no_robots (default-filters.php).
func TestFrontRobotsNoindexReplytocomP3143(t *testing.T) {
	c := testBoot(t)
	srv := &server.Server{WP: c, AbsPath: "/workspace/", Front: &server.Front{}}
	req := httptest.NewRequest(http.MethodGet, "/?p=1&replytocom=1", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	body := rec.Body.String()
	want := `content='noindex, follow, max-image-preview:large'`
	if !strings.Contains(body, want) {
		t.Fatalf("expected replytocom robots %q in %q", want, body[:min(800, len(body))])
	}
}

// P3.143: unapproved + moderation-hash → noindex.
func TestFrontRobotsNoindexUnapprovedPreviewP3143(t *testing.T) {
	c := testBoot(t)
	srv := &server.Server{WP: c, AbsPath: "/workspace/", Front: &server.Front{}}
	req := httptest.NewRequest(http.MethodGet, "/?p=1&unapproved=9&moderation-hash=abc", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	body := rec.Body.String()
	want := `content='noindex, follow, max-image-preview:large'`
	if !strings.Contains(body, want) {
		t.Fatalf("expected unapproved preview robots %q in %q", want, body[:min(800, len(body))])
	}
}

// P3.143: plain singular stays indexable (max-image-preview only when public).
func TestFrontRobotsPlainSingularNoForcedNoindexP3143(t *testing.T) {
	c := testBoot(t)
	srv := &server.Server{WP: c, AbsPath: "/workspace/", Front: &server.Front{}}
	req := httptest.NewRequest(http.MethodGet, "/?p=1", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	body := rec.Body.String()
	if strings.Contains(body, "noindex") {
		t.Fatalf("plain singular should not force noindex: %q", body[:min(800, len(body))])
	}
	if !strings.Contains(body, "max-image-preview:large") {
		t.Fatalf("expected max-image-preview on public singular: %q", body[:min(800, len(body))])
	}
}

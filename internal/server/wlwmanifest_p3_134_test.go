package server_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/server"
	"github.com/robertbaehr/GoPress/internal/wp"
)

// P3.134: wlwmanifest stub emits api.w.org Link and clears pingback headers.
func TestWLWManifestRestLinkP3134(t *testing.T) {
	c := testBoot(t)
	s := &server.Server{WP: c, AbsPath: "/workspace/", Front: &server.Front{}}
	req := httptest.NewRequest(http.MethodGet, "/wp-includes/wlwmanifest.xml", nil)
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d", rec.Code)
	}
	if rec.Header().Get("X-Pingback") != "" {
		t.Fatalf("X-Pingback not cleared: %q", rec.Header().Get("X-Pingback"))
	}
	link := strings.Join(rec.Header().Values("Link"), ",")
	if strings.Contains(link, "rel=\"pingback\"") || strings.Contains(link, "rel=pingback") {
		t.Fatalf("pingback Link remains: %q", link)
	}
	if !strings.Contains(link, "api.w.org") {
		t.Fatalf("expected api.w.org Link, got %q", link)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "clientType") {
		t.Fatalf("missing manifest body: %q", body)
	}
}

// P3.134: front head must not emit wlwmanifest link (WP 6.3 gone).
func TestFrontHeadNoWlwmanifestLinkP3134(t *testing.T) {
	c := testBoot(t)
	s := &server.Server{WP: c, AbsPath: "/workspace/", Front: &server.Front{}}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d", rec.Code)
	}
	body := rec.Body.String()
	if strings.Contains(body, "wlwmanifest") {
		t.Fatalf("front head must not emit wlwmanifest: %q", body[:min(800, len(body))])
	}
	if wp.WlwmanifestLink() != "" {
		t.Fatal("WlwmanifestLink must be empty (WP 6.3 gone)")
	}
}

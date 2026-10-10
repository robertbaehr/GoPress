package server_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/server"
	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestFeedLanguageFromLocale(t *testing.T) {
	c := testBoot(t)
	c.Locale = wp.NewLocale("de_DE")

	srv := &server.Server{WP: c, AbsPath: "/workspace/", Front: &server.Front{}}
	req := httptest.NewRequest(http.MethodGet, "/feed/", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	body := rec.Body.String()
	if !strings.Contains(body, "<language>de-DE</language>") {
		t.Fatalf("rss language missing de-DE: %q", body[:min(len(body), 500)])
	}

	req = httptest.NewRequest(http.MethodGet, "/feed/atom/", nil)
	rec = httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	body = rec.Body.String()
	if !strings.Contains(body, `xml:lang="de-DE"`) {
		t.Fatalf("atom xml:lang missing de-DE: %q", body[:min(len(body), 400)])
	}
}

func TestFeedLanguageDefaultEnUS(t *testing.T) {
	c := testBoot(t)
	// Fixture WPLANG empty → en-US (parity with WP determine_locale default).
	if got := wp.BlogInfoLanguage(c); got != "en-US" {
		t.Fatalf("fixture language=%q", got)
	}
	srv := &server.Server{WP: c, AbsPath: "/workspace/", Front: &server.Front{}}
	req := httptest.NewRequest(http.MethodGet, "/feed/", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if !strings.Contains(rec.Body.String(), "<language>en-US</language>") {
		t.Fatalf("expected en-US in feed")
	}
}

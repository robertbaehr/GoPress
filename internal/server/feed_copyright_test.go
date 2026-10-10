package server_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/server"
)

func TestFeedCopyrightEmittedWhenOptionSet(t *testing.T) {
	c := testBoot(t)
	t.Cleanup(func() { _, _ = c.Options.DeleteOption("rss_copyright") })
	if _, err := c.Options.UpdateOption("rss_copyright", "auto", "yes"); err != nil {
		t.Fatal(err)
	}

	srv := &server.Server{WP: c, AbsPath: "/workspace/", Front: &server.Front{}}
	req := httptest.NewRequest(http.MethodGet, "/feed/", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	body := rec.Body.String()
	if !strings.Contains(body, "<copyright>Copyright ") || !strings.Contains(body, "GoPress Fixture</copyright>") {
		t.Fatalf("rss missing copyright: %q", body[:min(len(body), 500)])
	}

	req = httptest.NewRequest(http.MethodGet, "/feed/atom/", nil)
	rec = httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	body = rec.Body.String()
	if !strings.Contains(body, "<rights>Copyright ") || !strings.Contains(body, "GoPress Fixture</rights>") {
		t.Fatalf("atom missing rights: %q", body[:min(len(body), 500)])
	}
}

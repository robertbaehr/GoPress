package server_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/server"
	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestFeedItemDCCreatorAppliesTheAuthorFilter(t *testing.T) {
	c := testBoot(t)
	if c.Hooks == nil {
		c.Hooks = wp.NewHooks()
	}
	c.Hooks.AddFilter("the_author", func(value any, _ ...any) any {
		if s, ok := value.(string); ok {
			return s + " (filtered)"
		}
		return value
	}, 10, 1)

	srv := &server.Server{WP: c, AbsPath: "/workspace/", Front: &server.Front{}}
	req := httptest.NewRequest(http.MethodGet, "/?feed=rss2", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "<dc:creator><![CDATA[Admin User (filtered)]]></dc:creator>") {
		t.Fatalf("expected filtered dc:creator, got %q", body[:min(900, len(body))])
	}
}

func TestFeedAtomAuthorAppliesTheAuthorFilter(t *testing.T) {
	c := testBoot(t)
	if c.Hooks == nil {
		c.Hooks = wp.NewHooks()
	}
	c.Hooks.AddFilter("the_author", func(value any, _ ...any) any {
		return "Atom Author"
	}, 10, 1)

	srv := &server.Server{WP: c, AbsPath: "/workspace/", Front: &server.Front{}}
	req := httptest.NewRequest(http.MethodGet, "/?feed=atom", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "<name>Atom Author</name>") {
		t.Fatalf("expected filtered Atom author name, got %q", body[:min(900, len(body))])
	}
}

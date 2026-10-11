package server_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/server"
)

func TestSingularCommentsFeedLinksExtraP3128(t *testing.T) {
	c := testBoot(t)
	srv := &server.Server{WP: c, AbsPath: "/workspace/", Front: &server.Front{}}

	// Hello world: comments open → should emit post comments feed.
	req := httptest.NewRequest(http.MethodGet, "/2024/06/15/hello-world/", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	body := rec.Body.String()
	if !strings.Contains(body, `Hello world! Comments Feed`) {
		t.Fatalf("expected singular comments feed: %q", body[:min(1500, len(body))])
	}

	// home-static: comments+pings closed, no comments → no singular comments feed.
	req = httptest.NewRequest(http.MethodGet, "/home-static/", nil)
	rec = httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	body = rec.Body.String()
	if strings.Contains(body, `Home Comments Feed`) {
		t.Fatalf("did not expect singular comments feed when closed: %q", body[:min(1500, len(body))])
	}
	// Global comments feed may still appear.
	if !strings.Contains(body, `Comments Feed`) {
		t.Fatalf("expected global comments feed still present")
	}
}

func TestSingularCommentsFeedFilterP3128(t *testing.T) {
	c := testBoot(t)
	c.Hooks.AddFilter("feed_links_extra_show_post_comments_feed", func(value any, _ ...any) any {
		return false
	}, 10, 1)
	srv := &server.Server{WP: c, AbsPath: "/workspace/", Front: &server.Front{}}
	req := httptest.NewRequest(http.MethodGet, "/2024/06/15/hello-world/", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	body := rec.Body.String()
	if strings.Contains(body, `Hello world! Comments Feed`) {
		t.Fatalf("filter should hide singular comments feed: %q", body[:min(1500, len(body))])
	}
}

package server_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/server"
	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestFeedItemSlashCommentsAppliesGetCommentsNumberFilter(t *testing.T) {
	c := testBoot(t)
	if c.Hooks == nil {
		c.Hooks = wp.NewHooks()
	}
	c.Hooks.AddFilter("get_comments_number", func(value any, _ ...any) any {
		return int64(42)
	}, 10, 2)

	srv := &server.Server{WP: c, AbsPath: "/workspace/", Front: &server.Front{}}
	req := httptest.NewRequest(http.MethodGet, "/?feed=rss2", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "<slash:comments>42</slash:comments>") {
		t.Fatalf("expected filtered slash:comments, got %q", body[:min(1200, len(body))])
	}
}

func TestFeedItemSlashCommentsPresent(t *testing.T) {
	c := testBoot(t)
	srv := &server.Server{WP: c, AbsPath: "/workspace/", Front: &server.Front{}}
	req := httptest.NewRequest(http.MethodGet, "/?feed=rss2", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `xmlns:slash="http://purl.org/rss/1.0/modules/slash/"`) {
		t.Fatalf("expected slash namespace: %q", body[:min(400, len(body))])
	}
	if !strings.Contains(body, "<slash:comments>") {
		t.Fatalf("expected slash:comments on open-comment posts: %q", body[:min(1200, len(body))])
	}
}

package server_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/server"
	"github.com/robertbaehr/GoPress/internal/wp"
)

// TestFrontFeedRSS2ItemSource covers P3.64: optional RSS <source> from post meta.
func TestFrontFeedRSS2ItemSource(t *testing.T) {
	c := testBoot(t)

	postID := int64(1) // Hello world
	_, ok, err := c.Meta.AddMetadata(wp.MetaPost, postID, "_rss_source_url", "https://example.com/feed/", true)
	if err != nil || !ok {
		t.Fatalf("AddMetadata url: ok=%v err=%v", ok, err)
	}
	_, ok, err = c.Meta.AddMetadata(wp.MetaPost, postID, "_rss_source_title", "Example Source", true)
	if err != nil || !ok {
		t.Fatalf("AddMetadata title: ok=%v err=%v", ok, err)
	}
	t.Cleanup(func() {
		_, _ = c.Meta.DeleteMetadata(wp.MetaPost, postID, "_rss_source_url", "https://example.com/feed/")
		_, _ = c.Meta.DeleteMetadata(wp.MetaPost, postID, "_rss_source_title", "Example Source")
	})

	srv := &server.Server{WP: c, AbsPath: "/workspace/", Front: &server.Front{}}
	req := httptest.NewRequest(http.MethodGet, "/?feed=rss2", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d", rec.Code)
	}
	body := rec.Body.String()
	want := `<source url="https://example.com/feed/">Example Source</source>`
	if !strings.Contains(body, want) {
		t.Fatalf("expected source tag %q in feed: %s", want, truncateFeed(body, 500))
	}
}

func TestFrontFeedRSS2ItemSourceAbsentByDefault(t *testing.T) {
	c := testBoot(t)
	srv := &server.Server{WP: c, AbsPath: "/workspace/", Front: &server.Front{}}
	req := httptest.NewRequest(http.MethodGet, "/?feed=rss2", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d", rec.Code)
	}
	if strings.Contains(rec.Body.String(), "<source ") {
		t.Fatal("fixture feed must not emit <source> without meta")
	}
}

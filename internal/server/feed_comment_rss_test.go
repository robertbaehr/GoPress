package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPostCommentsFeedLink(t *testing.T) {
	if got := postCommentsFeedLink("http://example.test/hello-world/", "rss2"); got != "http://example.test/hello-world/feed/" {
		t.Fatalf("rss2: %q", got)
	}
	if got := postCommentsFeedLink("http://example.test/hello-world", ""); got != "http://example.test/hello-world/feed/" {
		t.Fatalf("default: %q", got)
	}
	if got := postCommentsFeedLink("http://example.test/hello-world/", "atom"); got != "http://example.test/hello-world/feed/atom/" {
		t.Fatalf("atom: %q", got)
	}
	if postCommentsFeedLink("", "rss2") != "" {
		t.Fatal("empty permalink")
	}
}

func TestFeedItemCommentRss(t *testing.T) {
	c := testBootFeedCats(t)
	srv := &Server{WP: c, AbsPath: "/workspace/", Front: &Front{}}
	req := httptest.NewRequest(http.MethodGet, "/feed/", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "<wfw:commentRss>") || !strings.Contains(body, "</wfw:commentRss>") {
		t.Fatalf("expected wfw:commentRss on RSS items: %q", body[:min(900, len(body))])
	}
	// Pretty-permalink comments feed: …/hello-world/feed/
	if !strings.Contains(body, "/hello-world/feed/</wfw:commentRss>") {
		t.Fatalf("expected hello-world commentRss feed URL: %q", body[:min(1200, len(body))])
	}
	if !strings.Contains(body, "<slash:comments>") {
		t.Fatalf("expected slash:comments alongside commentRss: %q", body[:min(900, len(body))])
	}
}

func TestFeedAtomItemCommentsFeedLink(t *testing.T) {
	c := testBootFeedCats(t)
	srv := &Server{WP: c, AbsPath: "/workspace/", Front: &Front{}}
	req := httptest.NewRequest(http.MethodGet, "/feed/atom/", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `rel="replies" type="application/atom+xml"`) {
		t.Fatalf("expected atom comments feed replies link: %q", body[:min(900, len(body))])
	}
	if !strings.Contains(body, "/feed/atom/") {
		t.Fatalf("expected /feed/atom/ comments feed href: %q", body[:min(900, len(body))])
	}
}

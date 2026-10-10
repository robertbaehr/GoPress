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

func TestCommentsFeedRSS2GUIDAndReplyPrefix(t *testing.T) {
	c := testBootFeedCats(t)
	srv := &Server{WP: c, AbsPath: "/workspace/", Front: &Front{}}
	req := httptest.NewRequest(http.MethodGet, "/comments/feed/", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d", rec.Code)
	}
	body := rec.Body.String()
	// P3.77: get_comment_guid uses post GUID (pretty permalink for fixture-post-01).
	if !strings.Contains(body, "<guid isPermaLink=\"false\">http://127.0.0.1:8081/2024/02/01/fixture-post-01/#comment-9</guid>") {
		t.Fatalf("expected pretty post GUID comment guid: %q", body[:min(1800, len(body))])
	}
	// Child comment #4 → "In reply to …" in description + content:encoded.
	if !strings.Contains(body, "In reply to ") || !strings.Contains(body, "#comment-3") {
		t.Fatalf("expected In reply to parent link: %q", body[:min(2500, len(body))])
	}
	// content:encoded keeps real <a> for Gravatar in hello-world comment.
	if !strings.Contains(body, `<a href="https://gravatar.com/">Gravatar</a>`) {
		t.Fatalf("expected unescaped anchor in content:encoded: %q", body[:min(1200, len(body))])
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

package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFeedCommentsLinkHelper(t *testing.T) {
	c := testBootFeedCats(t)
	got := feedCommentsLink(c, "http://example.test/hello/", 0)
	if got != "http://example.test/hello/#respond" {
		t.Fatalf("zero comments: %q", got)
	}
	got = feedCommentsLink(c, "http://example.test/hello/", 2)
	if got != "http://example.test/hello/#comments" {
		t.Fatalf("with comments: %q", got)
	}
	if feedAtomCommentsHTMLLink("http://example.test/hello/") != "http://example.test/hello/#comments" {
		t.Fatal("atom link should always use #comments")
	}
}

func TestFeedRSSItemCommentsLink(t *testing.T) {
	c := testBootFeedCats(t)
	srv := &Server{WP: c, AbsPath: "/workspace/", Front: &Front{}}
	req := httptest.NewRequest(http.MethodGet, "/feed/", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "<comments>") || !strings.Contains(body, "#comments</comments>") {
		t.Fatalf("expected RSS item <comments> with #comments: %q", body[:min(800, len(body))])
	}
	if !strings.Contains(body, "#respond</comments>") {
		t.Fatalf("expected #respond for posts without comments: %q", body[:min(800, len(body))])
	}
}

func TestFeedAtomItemCommentsLink(t *testing.T) {
	c := testBootFeedCats(t)
	srv := &Server{WP: c, AbsPath: "/workspace/", Front: &Front{}}
	req := httptest.NewRequest(http.MethodGet, "/feed/atom/", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d", rec.Code)
	}
	body := rec.Body.String()
	if strings.Contains(body, `rel="replies" type="text/html"`) && strings.Contains(body, "#respond") {
		t.Fatalf("Atom HTML replies must use #comments not #respond: %q", body[:min(1200, len(body))])
	}
	if !strings.Contains(body, `rel="replies" type="text/html"`) || !strings.Contains(body, "#comments") {
		t.Fatalf("expected Atom HTML replies #comments: %q", body[:min(800, len(body))])
	}
}

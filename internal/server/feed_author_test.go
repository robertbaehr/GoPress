package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFeedAuthorChannelTitleAndItems(t *testing.T) {
	c := testBootFeedCats(t)
	srv := &Server{WP: c, AbsPath: "/workspace/", Front: &Front{}}
	req := httptest.NewRequest(http.MethodGet, "/author/admin/feed/", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String()[:min(400, rec.Body.Len())])
	}
	body := rec.Body.String()
	// wp_title_rss for author: "{Display Name} &#8211; {Blogname}"
	if !strings.Contains(body, "<title>Admin User &#8211; GoPress Fixture</title>") {
		t.Fatalf("expected author channel title, got %q", body[:min(600, len(body))])
	}
	if !strings.Contains(body, "<dc:creator><![CDATA[Admin User]]></dc:creator>") {
		t.Fatalf("expected item dc:creator, got %q", body[:min(800, len(body))])
	}
	if !strings.Contains(body, `rel="self"`) || !strings.Contains(body, "/author/admin/feed/") {
		t.Fatalf("expected self link for author feed: %q", body[:min(500, len(body))])
	}
}

func TestFeedAuthorHelperFallback(t *testing.T) {
	c := testBootFeedCats(t)
	name, url := feedAuthor(c, 1)
	if name != "Admin User" {
		t.Fatalf("name=%q", name)
	}
	if url == "" {
		t.Fatal("expected author URL")
	}
	name, _ = feedAuthor(c, 0)
	if name != "" {
		t.Fatalf("empty id should yield empty name, got %q", name)
	}
}

func TestFeedChannelTitleHelper(t *testing.T) {
	c := testBootFeedCats(t)
	got := feedChannelTitle(c, "GoPress Fixture", 1)
	if got != "Admin User &#8211; GoPress Fixture" {
		t.Fatalf("got %q", got)
	}
	got = feedChannelTitle(c, "GoPress Fixture", 0)
	if got != "GoPress Fixture" {
		t.Fatalf("site feed title=%q", got)
	}
}

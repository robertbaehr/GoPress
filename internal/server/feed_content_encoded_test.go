package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFeedRSSContentEncodedWhenAvailable(t *testing.T) {
	c := testBootFeedCats(t)
	srv := &Server{WP: c, AbsPath: "/workspace/", Front: &Front{}}
	req := httptest.NewRequest(http.MethodGet, "/feed/", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "<content:encoded>") {
		t.Fatal("expected content:encoded when rss_use_excerpt is off")
	}
	if !strings.Contains(body, "<description>") {
		t.Fatal("expected description")
	}
}

func TestFeedRSSOmitsContentEncodedWhenExcerptOnly(t *testing.T) {
	c := testBootFeedCats(t)
	if _, err := c.Options.UpdateOption("rss_use_excerpt", "1", "yes"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = c.Options.UpdateOption("rss_use_excerpt", "0", "yes")
	})
	srv := &Server{WP: c, AbsPath: "/workspace/", Front: &Front{}}
	req := httptest.NewRequest(http.MethodGet, "/feed/", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d", rec.Code)
	}
	body := rec.Body.String()
	if strings.Contains(body, "<content:encoded>") {
		t.Fatal("rss_use_excerpt=1 should omit content:encoded")
	}
	if !strings.Contains(body, "<description>") {
		t.Fatal("expected description still present")
	}
}

func TestRSSContentEncodedBodyHelpers(t *testing.T) {
	c := testBootFeedCats(t)
	p := feedPost{ExcerptRSS: "excerpt-only"}
	body, ok := rssContentEncodedBody(c, p)
	if !ok || body != "excerpt-only" {
		t.Fatalf("empty content → excerpt: ok=%v body=%q", ok, body)
	}
	p = feedPost{ContentRSS: "<p>Full</p>", ExcerptRSS: "excerpt"}
	body, ok = rssContentEncodedBody(c, p)
	if !ok || body != "<p>Full</p>" {
		t.Fatalf("prefer content: ok=%v body=%q", ok, body)
	}
	if _, err := c.Options.UpdateOption("rss_use_excerpt", "1", "yes"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = c.Options.UpdateOption("rss_use_excerpt", "0", "yes")
	})
	_, ok = rssContentEncodedBody(c, p)
	if ok {
		t.Fatal("rss_use_excerpt should omit")
	}
	tag := rss2ContentEncodedTag(c, p)
	if tag != "" {
		t.Fatalf("expected empty tag, got %q", tag)
	}
}

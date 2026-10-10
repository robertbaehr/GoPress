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

func TestTheExcerptRSSFilterOnDescription(t *testing.T) {
	c := testBootFeedCats(t)
	c.Hooks.AddFilter("the_excerpt_rss", func(value any, args ...any) any {
		s, _ := value.(string)
		return s + "<!--excerpt-rss-->"
	}, 10, 1)

	got := applyTheExcerptRSS(c, "Hello]]>World")
	if !strings.Contains(got, "]]&gt;") {
		t.Fatalf("expected CDATA escape, got %q", got)
	}
	if !strings.Contains(got, "<!--excerpt-rss-->") {
		t.Fatalf("expected the_excerpt_rss marker, got %q", got)
	}

	srv := &Server{WP: c, AbsPath: "/workspace/", Front: &Front{}}
	req := httptest.NewRequest(http.MethodGet, "/feed/", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "<!--excerpt-rss-->") {
		t.Fatal("expected the_excerpt_rss applied in /feed/ description")
	}
	// Filter should hit description CDATA, not only content:encoded.
	descIdx := strings.Index(body, "<description><![CDATA[")
	if descIdx < 0 || !strings.Contains(body[descIdx:descIdx+400], "<!--excerpt-rss-->") {
		t.Fatalf("marker missing near description")
	}
}

func TestTheContentFeedFilterOnContentEncoded(t *testing.T) {
	c := testBootFeedCats(t)
	c.Hooks.AddFilter("the_content_feed", func(value any, args ...any) any {
		s, _ := value.(string)
		feedType := ""
		if len(args) > 0 {
			feedType, _ = args[0].(string)
		}
		return s + "<!--feed:" + feedType + "-->"
	}, 10, 2)
	p := feedPost{ContentRSS: "<p>Hello]]>World</p>"}
	body, ok := rssContentEncodedBodyTyped(c, p, "rss2")
	if !ok {
		t.Fatal("expected body")
	}
	if !strings.Contains(body, "]]&gt;") {
		t.Fatalf("expected CDATA escape, got %q", body)
	}
	if strings.Contains(body, "]]>") {
		t.Fatalf("raw ]]> should be escaped, got %q", body)
	}
	if !strings.Contains(body, "<!--feed:rss2-->") {
		t.Fatalf("expected the_content_feed marker, got %q", body)
	}
	tag := rss2ContentEncodedTag(c, p)
	if !strings.Contains(tag, "<!--feed:rss2-->") {
		t.Fatalf("tag missing filter output: %q", tag)
	}

	srv := &Server{WP: c, AbsPath: "/workspace/", Front: &Front{}}
	req := httptest.NewRequest(http.MethodGet, "/feed/", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "<!--feed:rss2-->") {
		t.Fatal("expected the_content_feed applied in /feed/ body")
	}
}

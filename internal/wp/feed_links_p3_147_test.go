package wp_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// P3.147: FeedContentType maps rss2/atom/rdf; filter honored.
func TestFeedContentTypeP3147(t *testing.T) {
	c := testBootQuery(t)
	if got := wp.FeedContentType(c, ""); got != "application/rss+xml" {
		t.Fatalf("default=%q", got)
	}
	if got := wp.FeedContentType(c, "atom"); got != "application/atom+xml" {
		t.Fatalf("atom=%q", got)
	}
	if got := wp.FeedContentType(c, "rdf"); got != "application/rdf+xml" {
		t.Fatalf("rdf=%q", got)
	}
	c.Hooks.AddFilter("feed_content_type", func(value any, _ ...any) any {
		return "application/x-test+xml"
	}, 10, 2)
	if got := wp.FeedContentType(c, "rss2"); got != "application/x-test+xml" {
		t.Fatalf("filtered=%q", got)
	}
}

// P3.147: GetFeedLink pretty paths + feed_link filter.
func TestGetFeedLinkP3147(t *testing.T) {
	c := testBootQuery(t)
	home := "http://example.test"
	if got := wp.GetFeedLink(c, home, ""); got != "http://example.test/feed/" {
		t.Fatalf("default=%q", got)
	}
	if got := wp.GetFeedLink(c, home, "comments_rss2"); got != "http://example.test/comments/feed/" {
		t.Fatalf("comments=%q", got)
	}
	if got := wp.GetFeedLink(c, home, "atom"); got != "http://example.test/feed/atom/" {
		t.Fatalf("atom=%q", got)
	}
	c.Hooks.AddFilter("feed_link", func(value any, _ ...any) any {
		return "http://example.test/custom-feed/"
	}, 10, 2)
	if got := wp.GetFeedLink(c, home, ""); got != "http://example.test/custom-feed/" {
		t.Fatalf("filtered=%q", got)
	}
}

// P3.147: FeedAlternateLinkHTML EscURL-encodes href.
func TestFeedAlternateLinkHTMLEscURLP3147(t *testing.T) {
	c := testBootQuery(t)
	html := wp.FeedAlternateLinkHTML(c, "http://example.test", "", "Site &#187; Feed")
	for _, want := range []string{
		`rel="alternate"`,
		`type="application/rss+xml"`,
		`title="Site &#187; Feed"`,
		`href="http://example.test/feed/"`,
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("missing %q in %q", want, html)
		}
	}
	// Ampersand in home query should be EscURL-encoded in href.
	amp := wp.FeedAlternateLinkHTML(nil, "http://example.test/?x=1&y=2", "", "T")
	if strings.Contains(amp, `href="http://example.test/?x=1&y=2"`) {
		// EscURL may rewrite & → &#038;
		if !strings.Contains(amp, "&#038;") && !strings.Contains(amp, "&amp;") {
			t.Fatalf("expected EscURL amp encoding in %q", amp)
		}
	}
}

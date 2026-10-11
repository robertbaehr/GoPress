package wp_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// P3.148: GetPostCommentsFeedLink pretty paths + post_comments_feed_link filter.
func TestGetPostCommentsFeedLinkP3148(t *testing.T) {
	c := testBootQuery(t)
	perm := "http://example.test/hello-world/"
	if got := wp.GetPostCommentsFeedLink(c, perm, ""); got != "http://example.test/hello-world/feed/" {
		t.Fatalf("default=%q", got)
	}
	if got := wp.GetPostCommentsFeedLink(c, perm, "rss2"); got != "http://example.test/hello-world/feed/" {
		t.Fatalf("rss2=%q", got)
	}
	if got := wp.GetPostCommentsFeedLink(c, perm, "atom"); got != "http://example.test/hello-world/feed/atom/" {
		t.Fatalf("atom=%q", got)
	}
	if got := wp.GetPostCommentsFeedLink(c, "", "rss2"); got != "" {
		t.Fatalf("empty permalink=%q", got)
	}
	c.Hooks.AddFilter("post_comments_feed_link", func(value any, _ ...any) any {
		return "http://example.test/custom-comments-feed/"
	}, 10, 1)
	if got := wp.GetPostCommentsFeedLink(c, perm, "rss2"); got != "http://example.test/custom-comments-feed/" {
		t.Fatalf("filtered=%q", got)
	}
}

// P3.148: PostCommentsFeedLinkHTML EscURL-encodes href + filter.
func TestPostCommentsFeedLinkHTMLEscURLP3148(t *testing.T) {
	c := testBootQuery(t)
	html := wp.PostCommentsFeedLinkHTML(c, "http://example.test/p/", "rss2", "")
	for _, want := range []string{
		`<a href="http://example.test/p/feed/"`,
		`>Comments Feed</a>`,
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("missing %q in %q", want, html)
		}
	}
	amp := wp.PostCommentsFeedLinkHTML(nil, "http://example.test/?x=1&y=2", "rss2", "Feed")
	if strings.Contains(amp, `href="http://example.test/?x=1&y=2/feed/"`) {
		if !strings.Contains(amp, "&#038;") && !strings.Contains(amp, "&amp;") {
			t.Fatalf("expected EscURL amp encoding in %q", amp)
		}
	}
	c.Hooks.AddFilter("post_comments_feed_link_html", func(value any, _ ...any) any {
		return `<a href="/f/">Custom</a>`
	}, 10, 3)
	if got := wp.PostCommentsFeedLinkHTML(c, "http://example.test/p/", "rss2", "X"); got != `<a href="/f/">Custom</a>` {
		t.Fatalf("html filter=%q", got)
	}
}

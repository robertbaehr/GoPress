package wp_test

import (
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// P3.151: GetSearchLink / GetSearchFeedLink / GetSearchCommentsFeedLink pretty paths.
func TestGetSearchFeedLinkP3151(t *testing.T) {
	c := testBootQuery(t)
	home := "http://example.test"

	if got := wp.GetSearchLink(c, home, "hello world"); got != "http://example.test/search/hello+world/" {
		t.Fatalf("search link=%q", got)
	}
	if got := wp.GetSearchFeedLink(c, home, "gopress", ""); got != "http://example.test/search/gopress/feed/" {
		t.Fatalf("default feed=%q", got)
	}
	if got := wp.GetSearchFeedLink(c, home, "gopress", "atom"); got != "http://example.test/search/gopress/feed/atom/" {
		t.Fatalf("atom=%q", got)
	}
	comments := wp.GetSearchCommentsFeedLink(c, home, "gopress", "rss2")
	if comments != "http://example.test/search/gopress/feed/?withcomments=1" {
		t.Fatalf("comments feed=%q", comments)
	}

	c.Hooks.AddFilter("search_feed_link", func(value any, args ...any) any {
		feed, _ := args[0].(string)
		typ, _ := args[1].(string)
		if typ == "posts" && feed == "rss2" {
			return "http://example.test/custom-search-feed/"
		}
		if typ == "comments" {
			return "http://example.test/custom-search-comments-feed/"
		}
		return value
	}, 10, 3)
	if got := wp.GetSearchFeedLink(c, home, "x", "rss2"); got != "http://example.test/custom-search-feed/" {
		t.Fatalf("posts filtered=%q", got)
	}
	if got := wp.GetSearchCommentsFeedLink(c, home, "x", "rss2"); got != "http://example.test/custom-search-comments-feed/" {
		t.Fatalf("comments filtered=%q", got)
	}
}

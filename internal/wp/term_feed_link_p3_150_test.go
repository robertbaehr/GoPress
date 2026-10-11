package wp_test

import (
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// P3.150: GetCategoryFeedLink / GetTagFeedLink pretty paths + filters.
func TestGetTermFeedLinkP3150(t *testing.T) {
	c := testBootQuery(t)
	home := "http://example.test"

	cat := wp.GetCategoryFeedLink(c, home, 1, "")
	if cat != "http://example.test/category/uncategorized/feed/" {
		t.Fatalf("category default=%q", cat)
	}
	if got := wp.GetCategoryFeedLink(c, home, 1, "atom"); got != "http://example.test/category/uncategorized/feed/atom/" {
		t.Fatalf("category atom=%q", got)
	}
	tag := wp.GetTagFeedLink(c, home, 5, "rss2")
	if tag != "http://example.test/tag/fixture-tag-01/feed/" {
		t.Fatalf("tag=%q", tag)
	}
	if got := wp.GetTermFeedLink(c, home, 1, "category", "rss2"); got != cat {
		t.Fatalf("term alias=%q", got)
	}
	if got := wp.GetCategoryFeedLink(c, home, 0, "rss2"); got != "" {
		t.Fatalf("zero id=%q", got)
	}
	if got := wp.GetTagFeedLink(c, home, 1, "rss2"); got != "" {
		t.Fatalf("wrong taxonomy=%q", got)
	}

	c.Hooks.AddFilter("category_feed_link", func(value any, _ ...any) any {
		return "http://example.test/custom-cat-feed/"
	}, 10, 2)
	if got := wp.GetCategoryFeedLink(c, home, 1, "rss2"); got != "http://example.test/custom-cat-feed/" {
		t.Fatalf("category filtered=%q", got)
	}
	c.Hooks.AddFilter("tag_feed_link", func(value any, _ ...any) any {
		return "http://example.test/custom-tag-feed/"
	}, 10, 2)
	if got := wp.GetTagFeedLink(c, home, 5, "rss2"); got != "http://example.test/custom-tag-feed/" {
		t.Fatalf("tag filtered=%q", got)
	}
}

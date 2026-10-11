package wp_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// P3.149: GetAuthorPostsURL / GetAuthorFeedLink pretty paths + author_feed_link filter.
func TestGetAuthorFeedLinkP3149(t *testing.T) {
	c := testBootQuery(t)
	admin, err := c.GetUserBy("login", "admin")
	if err != nil || admin == nil {
		t.Fatalf("admin: %v", err)
	}
	home := "http://example.test"
	postsURL := wp.GetAuthorPostsURL(c, home, admin.ID)
	if postsURL == "" {
		t.Fatal("postsURL empty")
	}
	if !strings.HasPrefix(postsURL, "http://example.test/author/") || !strings.HasSuffix(postsURL, "/") {
		t.Fatalf("postsURL=%q", postsURL)
	}
	base := strings.TrimRight(postsURL, "/")
	if got := wp.GetAuthorFeedLink(c, home, admin.ID, ""); got != base+"/feed/" {
		t.Fatalf("default feed=%q want %q", got, base+"/feed/")
	}
	if got := wp.GetAuthorFeedLink(c, home, admin.ID, "atom"); got != base+"/feed/atom/" {
		t.Fatalf("atom=%q", got)
	}
	if got := wp.GetAuthorFeedLink(c, home, 0, "rss2"); got != "" {
		t.Fatalf("zero author=%q", got)
	}
	c.Hooks.AddFilter("author_feed_link", func(value any, _ ...any) any {
		return "http://example.test/custom-author-feed/"
	}, 10, 2)
	if got := wp.GetAuthorFeedLink(c, home, admin.ID, "rss2"); got != "http://example.test/custom-author-feed/" {
		t.Fatalf("filtered=%q", got)
	}
}

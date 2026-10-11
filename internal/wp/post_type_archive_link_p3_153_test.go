package wp_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// P3.153: post archive link is home when show_on_front != page.
func TestGetPostTypeArchiveLinkPostHomeP3153(t *testing.T) {
	c := testBootQuery(t)
	_, _ = c.Options.UpdateOption("show_on_front", "posts", "yes")
	got := wp.GetPostTypeArchiveLink(c, "post")
	if !strings.HasSuffix(got, "/") || got == "" {
		t.Fatalf("home archive=%q", got)
	}
	if wp.GetPostTypeArchiveLink(c, "book") != "" {
		t.Fatal("unknown type")
	}
	if wp.GetPostTypeArchiveLink(nil, "post") != "/" {
		t.Fatal("nil context home")
	}
}

// P3.153: feed link + filter.
func TestGetPostTypeArchiveFeedLinkP3153(t *testing.T) {
	c := testBootQuery(t)
	_, _ = c.Options.UpdateOption("show_on_front", "posts", "yes")
	got := wp.GetPostTypeArchiveFeedLink(c, "post", "")
	if !strings.HasSuffix(got, "/feed/") {
		t.Fatalf("rss2=%q", got)
	}
	atom := wp.GetPostTypeArchiveFeedLink(c, "post", "atom")
	if !strings.Contains(atom, "/feed/atom/") {
		t.Fatalf("atom=%q", atom)
	}
	c.Hooks.AddFilter("post_type_archive_feed_link", func(value any, _ ...any) any {
		return "https://filtered.example/feed/"
	}, 10, 1)
	if got := wp.GetPostTypeArchiveFeedLink(c, "post", "rss2"); got != "https://filtered.example/feed/" {
		t.Fatalf("filter=%q", got)
	}
}

func TestGetPostTypeArchiveLinkFilterP3153(t *testing.T) {
	c := testBootQuery(t)
	c.Hooks.AddFilter("post_type_archive_link", func(value any, args ...any) any {
		if args[0] != "post" {
			t.Fatalf("type=%v", args[0])
		}
		return "https://archive.example/blog/"
	}, 10, 2)
	if got := wp.GetPostTypeArchiveLink(c, "post"); got != "https://archive.example/blog/" {
		t.Fatalf("got=%q", got)
	}
}

package wp_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestBlogInfoNameDescriptionCharset(t *testing.T) {
	c := &wp.Context{Options: wp.NewOptions(nil, nil), Hooks: wp.NewHooks()}
	c.Options.Cache.Set("blogname", "Acme", "options")
	c.Options.Cache.Set("blogdescription", "Just another site", "options")
	c.Options.Cache.Set("blog_charset", "UTF-8", "options")
	c.Options.Cache.Set("home", "http://example.test", "options")
	c.Options.Cache.Set("siteurl", "http://example.test", "options")

	if got := wp.BlogInfo(c, "name", "raw"); got != "Acme" {
		t.Fatalf("name=%q", got)
	}
	if got := wp.BlogInfo(c, "description", "raw"); got != "Just another site" {
		t.Fatalf("description=%q", got)
	}
	if got := wp.BlogInfo(c, "charset", "raw"); got != "UTF-8" {
		t.Fatalf("charset=%q", got)
	}
	if got := wp.BlogInfo(c, "version", "raw"); got != wp.Version {
		t.Fatalf("version=%q", got)
	}
	if got := wp.BlogInfo(c, "language", "raw"); got != "en-US" {
		t.Fatalf("language=%q", got)
	}
	url := wp.BlogInfo(c, "url", "raw")
	if !strings.Contains(url, "example.test") {
		t.Fatalf("url=%q", url)
	}
	wpurl := wp.BlogInfo(c, "wpurl", "raw")
	if !strings.Contains(wpurl, "example.test") {
		t.Fatalf("wpurl=%q", wpurl)
	}
	feed := wp.BlogInfo(c, "rss2_url", "raw")
	if !strings.HasSuffix(feed, "/feed/") {
		t.Fatalf("rss2_url=%q", feed)
	}
}

func TestBlogInfoDisplayFilters(t *testing.T) {
	c := &wp.Context{Options: wp.NewOptions(nil, nil), Hooks: wp.NewHooks()}
	c.Options.Cache.Set("blogname", "Acme", "options")
	c.Options.Cache.Set("home", "http://example.test", "options")
	c.Hooks.AddFilter("bloginfo", func(value any, args ...any) any {
		return "Filtered"
	}, 10, 2)
	c.Hooks.AddFilter("bloginfo_url", func(value any, args ...any) any {
		return "http://filtered.test/"
	}, 10, 2)
	if got := wp.BlogInfo(c, "name", "display"); got != "Filtered" {
		t.Fatalf("display name=%q", got)
	}
	if got := wp.BlogInfo(c, "url", "display"); got != "http://filtered.test/" {
		t.Fatalf("display url=%q", got)
	}
	if got := wp.BlogInfo(c, "name", "raw"); got != "Acme" {
		t.Fatalf("raw name=%q", got)
	}
}

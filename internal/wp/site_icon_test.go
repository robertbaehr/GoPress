package wp_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestSiteIconRedirectURLDefault(t *testing.T) {
	c := bootWP(t)
	url := wp.SiteIconRedirectURL(c)
	if !strings.Contains(url, wp.DefaultSiteIconPath) {
		t.Fatalf("expected default logo path, got %q", url)
	}
	if wp.HasSiteIcon(c) {
		t.Fatal("fixture should not have a custom site icon")
	}
	if tags := wp.SiteIconMetaTags(c); tags != "" {
		t.Fatalf("expected empty meta tags without icon, got %q", tags)
	}
}

func TestSiteIconMetaTagsAndFilter(t *testing.T) {
	// P3.112: custom site_icon emits icon meta tags; get_site_icon_url filter applies.
	c := bootWP(t)
	orig := c.Options.GetOption("site_icon", "0")
	t.Cleanup(func() { _, _ = c.Options.UpdateOption("site_icon", orig, "yes") })
	if _, err := c.Options.UpdateOption("site_icon", "5", "yes"); err != nil {
		t.Fatal(err)
	}
	if !wp.HasSiteIcon(c) {
		t.Fatal("expected HasSiteIcon with attachment 5")
	}
	url512 := wp.GetSiteIconURL(c, 512, "")
	if url512 == "" {
		t.Fatal("expected site icon URL")
	}
	url32 := wp.GetSiteIconURL(c, 32, "")
	if url32 == "" {
		t.Fatal("expected 32px site icon URL")
	}
	tags := wp.SiteIconMetaTags(c)
	for _, want := range []string{
		`rel="icon"`,
		`sizes="32x32"`,
		`sizes="192x192"`,
		`rel="apple-touch-icon"`,
		`msapplication-TileImage`,
	} {
		if !strings.Contains(tags, want) {
			t.Fatalf("missing %q in %q", want, tags)
		}
	}
	c.Hooks.AddFilter("get_site_icon_url", func(value any, _ ...any) any {
		return "https://cdn.example/icon.png"
	}, 10, 3)
	if got := wp.GetSiteIconURL(c, 512, ""); got != "https://cdn.example/icon.png" {
		t.Fatalf("filter: %q", got)
	}
}

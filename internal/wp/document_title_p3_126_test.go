package wp_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestGetDocumentTitleFrontPage(t *testing.T) {
	c := &wp.Context{
		Hooks:   wp.NewHooks(),
		Options: wp.NewOptions(nil, nil),
		Scope:   &wp.RequestScope{IsFrontPage: true, IsHome: true},
	}
	c.Options.Cache.Set("blogname", "Acme", "options")
	c.Options.Cache.Set("blogdescription", "", "options")
	got := wp.GetDocumentTitle(c)
	if got != "Acme" {
		t.Fatalf("front title=%q want Acme", got)
	}
}

func TestGetDocumentTitlePreFilter(t *testing.T) {
	c := &wp.Context{
		Hooks: wp.NewHooks(),
		Scope: &wp.RequestScope{Is404: true},
	}
	c.Hooks.AddFilter("pre_get_document_title", func(value any, _ ...any) any {
		return "Short Circuit"
	}, 10, 1)
	if got := wp.GetDocumentTitle(c); got != "Short Circuit" {
		t.Fatalf("pre filter: %q", got)
	}
}

func TestGetDocumentTitleSearchAndSeparator(t *testing.T) {
	c := &wp.Context{
		Hooks:   wp.NewHooks(),
		Options: wp.NewOptions(nil, nil),
		Scope:   &wp.RequestScope{IsSearch: true, SearchQuery: "hello"},
	}
	c.Options.Cache.Set("blogname", "Site", "options")
	c.Hooks.AddFilter("document_title_separator", func(value any, _ ...any) any {
		return "|"
	}, 10, 1)
	got := wp.GetDocumentTitle(c)
	if !strings.Contains(got, "Search Results for") || !strings.Contains(got, "|") || !strings.Contains(got, "Site") {
		t.Fatalf("search title=%q", got)
	}
}

func TestGetDocumentTitleDocumentFilter(t *testing.T) {
	c := &wp.Context{
		Hooks:   wp.NewHooks(),
		Options: wp.NewOptions(nil, nil),
		Scope:   &wp.RequestScope{Is404: true},
	}
	c.Options.Cache.Set("blogname", "Site", "options")
	c.Hooks.AddFilter("document_title", func(value any, _ ...any) any {
		return "Filtered Title"
	}, 10, 1)
	if got := wp.GetDocumentTitle(c); got != "Filtered Title" {
		t.Fatalf("document_title filter: %q", got)
	}
}

func TestRenderTitleTagRequiresSupport(t *testing.T) {
	c := &wp.Context{
		Hooks:   wp.NewHooks(),
		Options: wp.NewOptions(nil, nil),
		Scope:   &wp.RequestScope{IsFrontPage: true},
	}
	c.Options.Cache.Set("blogname", "Acme", "options")
	if got := wp.RenderTitleTag(c); got != "" {
		t.Fatalf("without title-tag support want empty, got %q", got)
	}
	wp.AddThemeSupport(c, "title-tag")
	got := wp.RenderTitleTag(c)
	if !strings.HasPrefix(got, "<title>") || !strings.Contains(got, "Acme") {
		t.Fatalf("title tag: %q", got)
	}
}

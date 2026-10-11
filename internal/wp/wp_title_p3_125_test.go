package wp_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestGetWPTitleSingularRight(t *testing.T) {
	c := &wp.Context{
		Hooks: wp.NewHooks(),
		Scope: &wp.RequestScope{
			Is404: true,
		},
	}
	// 404 seeds a non-empty title → prefix/separator applied.
	got := wp.GetWPTitle(c, "|", "right")
	if !strings.Contains(got, "Page not found") || !strings.Contains(got, "|") {
		t.Fatalf("got %q", got)
	}
	if !strings.HasSuffix(strings.TrimSpace(got), "|") {
		t.Fatalf("expected trailing sep, got %q", got)
	}
}

func TestGetWPTitleSearchAnd404(t *testing.T) {
	c := &wp.Context{
		Hooks: wp.NewHooks(),
		Scope: &wp.RequestScope{IsSearch: true, SearchQuery: "gopher"},
	}
	got := wp.GetWPTitle(c, "»", "")
	if !strings.Contains(got, "Search Results") || !strings.Contains(got, "gopher") {
		t.Fatalf("search title=%q", got)
	}

	c.Scope = &wp.RequestScope{Is404: true}
	got = wp.GetWPTitle(c, "»", "right")
	if !strings.Contains(got, "Page not found") {
		t.Fatalf("404 title=%q", got)
	}
}

func TestGetWPTitleFilter(t *testing.T) {
	c := &wp.Context{
		Hooks: wp.NewHooks(),
		Scope: &wp.RequestScope{Is404: true},
	}
	c.Hooks.AddFilter("wp_title", func(value any, args ...any) any {
		return "Custom"
	}, 10, 3)
	if got := wp.GetWPTitle(c, "|", "right"); got != "Custom" {
		t.Fatalf("got %q", got)
	}
}

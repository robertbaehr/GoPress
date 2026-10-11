package wp_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// P2.140: non-empty text passes through wp_trim_excerpt filter unchanged.
func TestWPTrimExcerptPassthroughP2140(t *testing.T) {
	c := &wp.Context{Hooks: wp.NewHooks()}
	got := wp.WPTrimExcerpt(c, "Keep me", &wp.Post{Content: "ignored body"})
	if got != "Keep me" {
		t.Fatalf("passthrough: %q", got)
	}
	c.Hooks.AddFilter("wp_trim_excerpt", func(value any, args ...any) any {
		return "filtered-trim"
	}, 10, 2)
	got = wp.WPTrimExcerpt(c, "Keep me", nil)
	if got != "filtered-trim" {
		t.Fatalf("filter: %q", got)
	}
}

// P2.140: empty text generates from content with excerpt_length / excerpt_more.
func TestWPTrimExcerptFromContentP2140(t *testing.T) {
	c := &wp.Context{Hooks: wp.NewHooks()}
	words := strings.Repeat("word ", 60)
	post := &wp.Post{ID: 1, Content: "<!-- wp:paragraph --><p>" + words + "</p><!-- /wp:paragraph -->"}
	got := wp.WPTrimExcerpt(c, "", post)
	if strings.Contains(got, "<p>") || strings.Contains(got, "<!--") {
		t.Fatalf("should strip tags/comments: %q", got)
	}
	if !strings.HasSuffix(got, "[&hellip;]") {
		t.Fatalf("expected excerpt_more suffix: %q", got)
	}
	trimmed := strings.TrimSuffix(strings.TrimSpace(got), "[&hellip;]")
	trimmed = strings.TrimSpace(trimmed)
	if n := len(strings.Fields(trimmed)); n != 55 {
		t.Fatalf("word count=%d got=%q", n, got)
	}
}

func TestWPTrimExcerptFiltersP2140(t *testing.T) {
	c := &wp.Context{Hooks: wp.NewHooks()}
	c.Hooks.AddFilter("excerpt_length", func(value any, _ ...any) any {
		return 3
	}, 10, 1)
	c.Hooks.AddFilter("excerpt_more", func(value any, _ ...any) any {
		return " >>"
	}, 10, 1)
	got := wp.WPTrimExcerpt(c, "", &wp.Post{Content: "one two three four five"})
	if got != "one two three >>" {
		t.Fatalf("filters: %q", got)
	}
}

func TestGetTheExcerptUsesWPTrimExcerptP2140(t *testing.T) {
	c := &wp.Context{Hooks: wp.NewHooks()}
	words := strings.Repeat("alpha ", 60)
	got := wp.GetTheExcerpt(c, &wp.Post{Content: "<p>" + words + "</p>"})
	if !strings.Contains(got, "[&hellip;]") {
		t.Fatalf("GetTheExcerpt should use wp_trim_excerpt more: %q", got)
	}
}

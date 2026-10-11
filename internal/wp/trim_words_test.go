package wp_test

import (
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestTrimWordsBasic(t *testing.T) {
	got := wp.TrimWords("one two three four five six", 5, "&hellip;")
	want := "one two three four five&hellip;"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
	if g := wp.TrimWords("one two", 5, "&hellip;"); g != "one two" {
		t.Fatalf("no trim: %q", g)
	}
	if g := wp.TrimWordsEllipsis("a b c d e f", 3); g != "a b c&hellip;" {
		t.Fatalf("ellipsis: %q", g)
	}
}

func TestTrimWordsStripsTagsAndWhitespace(t *testing.T) {
	got := wp.TrimWords("<p>hi</p>\nthere\tfriend", 10, "&hellip;")
	if got != "hi there friend" {
		t.Fatalf("got %q", got)
	}
	if g := wp.TrimWords("<script>alert(1)</script>", 5, "&hellip;"); g != "" {
		t.Fatalf("script strip: %q", g)
	}
}

func TestTrimWordsFiltered(t *testing.T) {
	c := bootWP(t)
	c.Hooks.AddFilter("wp_trim_words", func(value any, args ...any) any {
		return "FILTERED"
	}, 10, 4)
	got := wp.TrimWordsFiltered(c, "one two three four", 2, "…")
	if got != "FILTERED" {
		t.Fatalf("filter: %q", got)
	}
}

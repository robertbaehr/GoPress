package wp_test

import (
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// P2.139: wp_html_excerpt strips tags, truncates by rune, appends more.
func TestWPHtmlExcerptP2139(t *testing.T) {
	got := wp.WPHtmlExcerpt(`<p onclick="x">hello world</p>`, 5, "&hellip;")
	if got != "hello&hellip;" {
		t.Fatalf("got %q want hello&hellip;", got)
	}
	got = wp.WPHtmlExcerpt(`<script>alert(1)</script>`, 40, "&hellip;")
	if got != "" {
		t.Fatalf("script strip: %q", got)
	}
	// Incomplete entity at cut point is removed before more.
	// "Tom &amp; Jerry" → substr(6)="Tom &a" → strip "&a" → "Tom " → "Tom…"
	got = wp.WPHtmlExcerpt(`Tom &amp; Jerry`, 6, "…")
	if got != "Tom…" {
		t.Fatalf("entity trim: %q want Tom…", got)
	}
	// No truncation → no more suffix.
	got = wp.WPHtmlExcerpt("hi", 40, "&hellip;")
	if got != "hi" {
		t.Fatalf("short: %q", got)
	}
	// Null more defaults to empty in PHP; we pass "".
	got = wp.WPHtmlExcerpt("abcdefghij", 4, "")
	if got != "abcd" {
		t.Fatalf("empty more: %q", got)
	}
}

func TestWPHtmlExcerptGoldenSubsetP2139(t *testing.T) {
	vectors := loadFormattingVectors(t)
	for i, v := range vectors {
		if v.WPHtmlExcerpt == "" && v.Input == "" {
			continue
		}
		got := wp.WPHtmlExcerptEllipsis(v.Input, 40)
		if got != v.WPHtmlExcerpt {
			t.Fatalf("vector[%d] input=%q got=%q want=%q", i, v.Input, got, v.WPHtmlExcerpt)
		}
	}
}

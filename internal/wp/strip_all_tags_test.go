package wp_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestWPStripAllTagsBasic(t *testing.T) {
	cases := []struct {
		in, want string
		breaks   bool
	}{
		{"", "", false},
		{"plain", "plain", false},
		{"<b>bold</b>", "bold", false},
		{"a <i>b</i> c", "a b c", false},
		{"<script>alert(1)</script>hi", "hi", false},
		{"<SCRIPT>alert(1)</SCRIPT>hi", "hi", false},
		{"<style>.x{color:red}</style>ok", "ok", false},
		{"<p>one</p>\n<p>two</p>", "one\ntwo", false},
		{"<p>one</p>\n<p>two</p>", "one two", true},
		{"  spaced   \t\n  ", "spaced", true},
		{"<a href=\"x\">link</a>", "link", false},
		{"before<!-- comment -->after", "beforeafter", false},
		{"x\x00y", "xy", false},
		{"<div><span>nest</span></div>", "nest", false},
		{"keep <br> breaks", "keep  breaks", false},
		{"a\tb\nc", "a b c", true},
	}
	for _, tc := range cases {
		got := wp.WPStripAllTags(tc.in, tc.breaks)
		if got != tc.want {
			t.Fatalf("WPStripAllTags(%q, %v)=%q want %q", tc.in, tc.breaks, got, tc.want)
		}
	}
}

func TestWPStripAllTagsScriptStyleBackref(t *testing.T) {
	// Proper pair still removes body (PHP \\1 backref).
	if got := wp.WPStripAllTags(`<script>evil</script>safe`, false); got != "safe" {
		t.Fatalf("script pair: got %q", got)
	}
	if got := wp.WPStripAllTags(`<style>.x{}</style>safe`, false); got != "safe" {
		t.Fatalf("style pair: got %q", got)
	}
	// Mismatched closers: script opener closed by </style> must not wipe via backref.
	got := wp.WPStripAllTags(`<script>keep</style>yes`, false)
	if !strings.Contains(got, "yes") {
		t.Fatalf("mismatched closer wiped too much: %q", got)
	}
}

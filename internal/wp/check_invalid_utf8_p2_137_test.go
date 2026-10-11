package wp_test

import (
	"testing"
	"unicode/utf8"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// P2.137: is_utf8_charset / _is_utf8_charset.
func TestIsUTF8Charset(t *testing.T) {
	cases := []struct {
		in   any
		want bool
	}{
		{"UTF-8", true},
		{"utf-8", true},
		{"UTF8", true},
		{"utf8", true},
		{"ISO-8859-1", false},
		{"latin1", false},
		{" UTF-8 ", false}, // WP strcasecmp — no trim
		{"", false},
		{nil, false},
		{123, false},
	}
	for _, tc := range cases {
		if got := wp.IsUTF8Charset(tc.in); got != tc.want {
			t.Fatalf("IsUTF8Charset(%v)=%v want %v", tc.in, got, tc.want)
		}
	}
}

// P2.137: non-UTF-8 charset skips validation (passthrough).
func TestCheckInvalidUTF8ForCharsetPassthrough(t *testing.T) {
	invalid := "the byte \xC0 is never allowed"
	if got := wp.CheckInvalidUTF8ForCharset(invalid, false, "ISO-8859-1"); got != invalid {
		t.Fatalf("ISO-8859-1 passthrough: got %q", got)
	}
	if got := wp.CheckInvalidUTF8ForCharset(invalid, true, "latin1"); got != invalid {
		t.Fatalf("latin1 strip ignored: got %q", got)
	}
	if got := wp.CheckInvalidUTF8ForCharset(invalid, false, "UTF-8"); got != "" {
		t.Fatalf("UTF-8 reject: got %q", got)
	}
	want := "the byte " + string(utf8.RuneError) + " is never allowed"
	if got := wp.CheckInvalidUTF8ForCharset(invalid, true, "utf8"); got != want {
		t.Fatalf("utf8 strip: got %q want %q", got, want)
	}
}

// P2.137: CheckInvalidUTF8Ctx honors blog_charset.
func TestCheckInvalidUTF8CtxBlogCharset(t *testing.T) {
	invalid := "bad \xC0 byte"
	c := &wp.Context{Options: wp.NewOptions(nil, wp.NewObjectCache()), Hooks: wp.NewHooks()}

	// Default / missing → UTF-8 → reject.
	if got := wp.CheckInvalidUTF8Ctx(c, invalid, false); got != "" {
		t.Fatalf("default charset reject: got %q", got)
	}
	if !wp.BlogIsUTF8Charset(c) {
		t.Fatal("default BlogIsUTF8Charset want true")
	}

	c.Options.Cache.Set("blog_charset", "ISO-8859-1", "options")
	if wp.BlogIsUTF8Charset(c) {
		t.Fatal("ISO-8859-1 BlogIsUTF8Charset want false")
	}
	if got := wp.CheckInvalidUTF8Ctx(c, invalid, false); got != invalid {
		t.Fatalf("ISO-8859-1 ctx passthrough: got %q", got)
	}

	c.Options.Cache.Set("blog_charset", "utf-8", "options")
	if !wp.BlogIsUTF8Charset(c) {
		t.Fatal("utf-8 BlogIsUTF8Charset want true")
	}
	if got := wp.CheckInvalidUTF8Ctx(c, invalid, false); got != "" {
		t.Fatalf("utf-8 ctx reject: got %q", got)
	}

	// Nil context defaults to UTF-8 validation.
	if got := wp.CheckInvalidUTF8Ctx(nil, invalid, false); got != "" {
		t.Fatalf("nil ctx reject: got %q", got)
	}
}

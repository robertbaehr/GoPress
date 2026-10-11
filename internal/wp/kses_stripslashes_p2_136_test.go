package wp_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// P2.136: wp_kses_stripslashes converts \" to ".
func TestKsesStripslashesP2136(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{``, ``},
		{`plain`, `plain`},
		{`href=\"http://x\"`, `href="http://x"`},
		{`<a title=\"hi\">`, `<a title="hi">`},
		{`no escapes "here"`, `no escapes "here"`},
		{`keep \\ path`, `keep \\ path`},
	}
	for _, tc := range cases {
		if got := wp.KsesStripslashes(tc.in); got != tc.want {
			t.Fatalf("KsesStripslashes(%q)=%q want %q", tc.in, got, tc.want)
		}
	}
}

// P2.136: KsesPost path preserves attributes after stripslashes of \" in attr values.
func TestKsesStripslashesWiredIntoSplitP2136(t *testing.T) {
	// Attribute value with escaped quotes as WP magic-quotes-era chunks may present.
	in := `<p title=\"Hello\">ok</p>`
	out := wp.KsesPost(in)
	if !strings.Contains(out, `title="Hello"`) && !strings.Contains(out, `title=\"Hello\"`) {
		// After stripslashes, title should be a normal quoted attr or kept safely.
		t.Fatalf("unexpected kses output: %q", out)
	}
	if strings.Contains(out, `title=\"Hello\"`) {
		t.Fatalf("expected stripslashes to remove backslash-quotes: %q", out)
	}
	if !strings.Contains(out, "ok") {
		t.Fatalf("missing inner text: %q", out)
	}
}

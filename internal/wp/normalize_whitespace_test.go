package wp_test

import (
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestNormalizeWhitespace(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"", ""},
		{"  spaced  ", "spaced"},
		{"a\tb", "a b"},
		{"a  \t  b", "a b"},
		{"line1\nline2", "line1\nline2"},
		{"line1\n\n\nline2", "line1\nline2"},
		{"line1\r\nline2", "line1\nline2"},
		{"line1\r\rline2", "line1\nline2"},
		{"  a\n\n  b  \t c  ", "a\n b c"},
	}
	for _, tc := range cases {
		if got := wp.NormalizeWhitespace(tc.in); got != tc.want {
			t.Fatalf("NormalizeWhitespace(%q)=%q want %q", tc.in, got, tc.want)
		}
	}
}

package wp_test

import (
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestUTF8URIEncodeP2144(t *testing.T) {
	cases := []struct {
		in          string
		length      int
		encodeASCII bool
		want        string
	}{
		{"café", 0, false, "caf%c3%a9"},
		{"café", 0, true, "caf%c3%a9"},
		{"hello world", 0, true, "hello%20world"},
		{"a b", 0, true, "a%20b"},
		{"«quotes»", 200, false, "%c2%abquotes%c2%bb"},
		{"abcdef", 3, false, "abc"},
		{"日本語", 6, false, ""},
		{"日本語", 9, false, "%e6%97%a5"},
		{"", 0, false, ""},
		{"plain", 0, false, "plain"},
	}
	for _, tc := range cases {
		got := wp.UTF8URIEncode(tc.in, tc.length, tc.encodeASCII)
		if got != tc.want {
			t.Fatalf("UTF8URIEncode(%q,%d,%v)=%q want %q", tc.in, tc.length, tc.encodeASCII, got, tc.want)
		}
		if got2 := wp.WPUTF8URIEncode(tc.in, tc.length, tc.encodeASCII); got2 != tc.want {
			t.Fatalf("WPUTF8URIEncode mismatch: %q", got2)
		}
	}
}

func TestUTF8URIEncodeMatchesSanitizeTitleDashesP2144(t *testing.T) {
	// Display context keeps encoded angle quotes (via utf8_uri_encode).
	if got := wp.SanitizeTitleWithDashes("«quotes»", "", "display"); got != "%c2%abquotes%c2%bb" {
		t.Fatalf("display=%q", got)
	}
}

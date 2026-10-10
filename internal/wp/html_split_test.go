package wp_test

import (
	"reflect"
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestWPHTMLSplitBasic(t *testing.T) {
	cases := []struct {
		in   string
		want []string
	}{
		{"", []string{""}},
		{"plain", []string{"plain"}},
		{"a<b>c", []string{"a", "<b>", "c"}},
		{"<p>hi</p>", []string{"", "<p>", "hi", "</p>", ""}},
		{"x<!--c-->y", []string{"x", "<!--c-->", "y"}},
		{"a<b><c>d", []string{"a", "<b>", "", "<c>", "d"}},
		{"<![CDATA[x]]>z", []string{"", "<![CDATA[x]]>", "z"}},
		{"<!--unclosed", []string{"", "<!--unclosed", ""}},
		{"<br", []string{"", "<br", ""}},
	}
	for _, tc := range cases {
		got := wp.WPHTMLSplit(tc.in)
		if !reflect.DeepEqual(got, tc.want) {
			t.Fatalf("WPHTMLSplit(%q)=%#v want %#v", tc.in, got, tc.want)
		}
	}
}

func TestWPHTMLSplitRoundtrip(t *testing.T) {
	in := `<div class="x">a<!-- hi -->b</div>`
	parts := wp.WPHTMLSplit(in)
	var b string
	for _, p := range parts {
		b += p
	}
	if b != in {
		t.Fatalf("roundtrip %q → %q parts=%#v", in, b, parts)
	}
}

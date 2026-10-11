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

func TestWPHTMLSplitQuotedGreaterThan(t *testing.T) {
	// P2.109: '>' inside quoted attrs must not end the tag.
	in := `x<img alt="a>b" src='c>d'>y`
	got := wp.WPHTMLSplit(in)
	want := []string{"x", `<img alt="a>b" src='c>d'>`, "y"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("WPHTMLSplit(%q)=%#v want %#v", in, got, want)
	}
	in2 := `<span title="1>2">ok</span>`
	got2 := wp.WPHTMLSplit(in2)
	want2 := []string{"", `<span title="1>2">`, "ok", "</span>", ""}
	if !reflect.DeepEqual(got2, want2) {
		t.Fatalf("WPHTMLSplit(%q)=%#v want %#v", in2, got2, want2)
	}
}

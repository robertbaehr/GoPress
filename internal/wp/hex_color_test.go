package wp_test

import (
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestIsHexColor(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"#abc", true},
		{"#ABC", true},
		{"#aabbcc", true},
		{"#AAbbCC", true},
		{"#fff", true},
		{"#000000", true},
		{"", false},
		{"abc", false},
		{"#gg0000", false},
		{"#abcd", false},
		{"#aabbccd", false},
		{"rgb(0,0,0)", false},
		{" #abc ", true},
	}
	for _, tc := range cases {
		if got := wp.IsHexColor(tc.in); got != tc.want {
			t.Fatalf("IsHexColor(%q)=%v want %v", tc.in, got, tc.want)
		}
	}
}

func TestSanitizeHexColorFamily(t *testing.T) {
	if got := wp.SanitizeHexColor("#1a2b3c"); got != "#1a2b3c" {
		t.Fatalf("SanitizeHexColor=%q", got)
	}
	if got := wp.SanitizeHexColor("nope"); got != "" {
		t.Fatalf("invalid SanitizeHexColor=%q", got)
	}
	if got := wp.SanitizeHexColorNoHash("1a2b3c"); got != "1a2b3c" {
		t.Fatalf("SanitizeHexColorNoHash=%q", got)
	}
	if got := wp.SanitizeHexColorNoHash("#abc"); got != "abc" {
		t.Fatalf("SanitizeHexColorNoHash hashed=%q", got)
	}
	if got := wp.MaybeHashHexColor("abc"); got != "#abc" {
		t.Fatalf("MaybeHashHexColor=%q", got)
	}
	if got := wp.MaybeHashHexColor("not-a-color"); got != "not-a-color" {
		t.Fatalf("MaybeHashHexColor passthrough=%q", got)
	}
}

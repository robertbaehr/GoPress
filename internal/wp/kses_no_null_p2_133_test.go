package wp_test

import (
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// P2.133: KsesNoNull strips C0 controls and slash-zero runs by default.
func TestKsesNoNullP2133(t *testing.T) {
	in := "a\x00b\x07c\td\ne"
	got := wp.KsesNoNull(in, nil)
	want := "abc\td\ne"
	if got != want {
		t.Fatalf("controls: got %q want %q", got, want)
	}
	if got := wp.KsesNoNull(`hello\0world`, nil); got != "helloworld" {
		t.Fatalf("slash-zero remove: %q", got)
	}
	if got := wp.KsesNoNull(`x\\0y`, nil); got != "xy" {
		t.Fatalf("multi-slash zero: %q", got)
	}
	if got := wp.KsesNoNull(`keep\0me`, &wp.KsesNoNullOptions{SlashZero: "keep"}); got != `keep\0me` {
		t.Fatalf("slash-zero keep: %q", got)
	}
	// keep still strips raw NUL bytes.
	if got := wp.KsesNoNull("a\x00b", &wp.KsesNoNullOptions{SlashZero: "keep"}); got != "ab" {
		t.Fatalf("keep strips NUL: %q", got)
	}
}

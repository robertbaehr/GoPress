package wp_test

import (
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// P2.132: KsesBadProtocol mirrors wp_kses_bad_protocol allowlist behavior.
func TestKsesBadProtocolP2132(t *testing.T) {
	if got := wp.KsesBadProtocol("https://example.com/a", nil); got != "https://example.com/a" {
		t.Fatalf("https short-circuit: %q", got)
	}
	if got := wp.KsesBadProtocol("http://example.com/a", []string{"http", "https"}); got != "http://example.com/a" {
		t.Fatalf("http allowed: %q", got)
	}
	if got := wp.KsesBadProtocol("javascript:alert(1)", nil); got != "alert(1)" {
		t.Fatalf("strip javascript: %q", got)
	}
	if got := wp.KsesBadProtocol("javascript:alert(1)", []string{"https"}); got != "alert(1)" {
		t.Fatalf("custom allowlist strips js: %q", got)
	}
	if got := wp.KsesBadProtocol("https://x.test", []string{"http"}); got != "//x.test" {
		t.Fatalf("https denied when only http allowed: %q", got)
	}
	if got := wp.KsesBadProtocol("mailto:a@b.c", []string{"mailto"}); got != "mailto:a@b.c" {
		t.Fatalf("mailto allowed: %q", got)
	}
}

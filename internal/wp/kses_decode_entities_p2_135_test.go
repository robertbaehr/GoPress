package wp_test

import (
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// P2.135: KsesDecodeEntities expands decimal and hex numeric character refs.
func TestKsesDecodeEntitiesP2135(t *testing.T) {
	if got := wp.KsesDecodeEntities("jav&#97;script"); got != "javascript" {
		t.Fatalf("decimal: %q", got)
	}
	if got := wp.KsesDecodeEntities("jav&#x61;script"); got != "javascript" {
		t.Fatalf("hex lower: %q", got)
	}
	if got := wp.KsesDecodeEntities("jav&#X61;script"); got != "javascript" {
		t.Fatalf("hex upper: %q", got)
	}
	if got := wp.KsesDecodeEntities("plain"); got != "plain" {
		t.Fatalf("noop: %q", got)
	}
	// PHP chr wraps >255 with modulo 256.
	if got := wp.KsesDecodeEntities("&#256;"); got != string(rune(0)) {
		t.Fatalf("mod 256: %q", got)
	}
}

// P2.135: bad-protocol path decodes entities in the scheme before allowlist check.
func TestKsesBadProtocolDecodesEntitiesP2135(t *testing.T) {
	got := wp.KsesBadProtocol("jav&#97;script:alert(1)", nil)
	if got != "alert(1)" {
		t.Fatalf("entity-encoded javascript: %q", got)
	}
	got = wp.KsesBadProtocol("htt&#112;s://example.com/a", []string{"https"})
	if got != "https://example.com/a" {
		t.Fatalf("entity-encoded https allowed: %q", got)
	}
}

package wp_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestKsesOneAttrKeepsAllowed(t *testing.T) {
	out := wp.KsesOneAttr(`href="https://example.com"`, "a")
	if !strings.Contains(out, `href=`) || !strings.Contains(out, "example.com") {
		t.Fatalf("got %q", out)
	}
}

func TestKsesOneAttrRejectsDisallowed(t *testing.T) {
	out := wp.KsesOneAttr(`onclick="alert(1)"`, "a")
	if strings.TrimSpace(out) != "" {
		t.Fatalf("expected empty, got %q", out)
	}
}

func TestKsesOneAttrPreservesWhitespace(t *testing.T) {
	out := wp.KsesOneAttr("  class=\"btn\"  ", "a")
	if !strings.HasPrefix(out, "  ") || !strings.HasSuffix(out, "  ") {
		t.Fatalf("whitespace not preserved: %q", out)
	}
	if !strings.Contains(out, `class=`) {
		t.Fatalf("missing class: %q", out)
	}
}

func TestKsesOneAttrURIProtocol(t *testing.T) {
	out := wp.KsesOneAttr(`href="javascript:alert(1)"`, "a")
	// Bad protocol stripped; attribute may remain empty or protocol-less.
	if strings.Contains(strings.ToLower(out), "javascript:") {
		t.Fatalf("javascript protocol leaked: %q", out)
	}
}

func TestKsesOneAttrValueless(t *testing.T) {
	out := wp.KsesOneAttr("disabled", "input")
	// input may or may not allow disabled in post context; just ensure no panic.
	_ = out
	out = wp.KsesOneAttr("download", "a")
	if strings.TrimSpace(out) == "" {
		// download is allowed on a in post HTML
		t.Fatalf("expected download kept: %q", out)
	}
}

package wp_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestKsesSplitAllowsAndStripsP2130(t *testing.T) {
	allowed := wp.AllowedHTML{
		"a":      {"href": true, "title": true},
		"p":      {},
		"em":     {},
		"strong": {},
	}
	in := `<p>Hi <a href="https://example.com" onclick="alert(1)">link</a> <script>x</script><em>ok</em></p>`
	out := wp.KsesSplit(in, allowed, nil)
	if !strings.Contains(out, `<a href="https://example.com"`) {
		t.Fatalf("expected kept href: %q", out)
	}
	if strings.Contains(out, "onclick") || strings.Contains(out, "<script") {
		t.Fatalf("expected strip onclick/script: %q", out)
	}
	if !strings.Contains(out, "<em>ok</em>") {
		t.Fatalf("expected em kept: %q", out)
	}
}

func TestKsesUsesKsesSplitP2130(t *testing.T) {
	allowed := wp.AllowedHTML{"b": {}}
	in := `<b>bold</b><i>italic</i>`
	viaSplit := wp.KsesSplit(in, allowed, nil)
	viaKses := wp.Kses(in, allowed, nil)
	if viaSplit != viaKses {
		t.Fatalf("Kses/KsesSplit mismatch:\n split=%q\n kses=%q", viaSplit, viaKses)
	}
	if strings.Contains(viaKses, "<i>") {
		t.Fatalf("expected i stripped: %q", viaKses)
	}
}

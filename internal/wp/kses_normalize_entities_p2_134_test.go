package wp_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// P2.134: KsesNormalizeEntities restores valid refs and keeps invalid disarmed.
func TestKsesNormalizeEntitiesP2134(t *testing.T) {
	got := wp.KsesNormalizeEntities(`a & b &amp; &copy; &#38; &#x26; &notanentity;`, "html")
	for _, want := range []string{
		`a &amp; b`,
		`&amp;`,
		`&copy;`,
		`&#038;`,
		`&#x26;`,
		`&amp;notanentity;`,
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %q in %q", want, got)
		}
	}
	// Double-encoded numeric stays distinct from single after normalize order.
	got = wp.KsesNormalizeEntities(`&#x2E;`, "html")
	if got != `&#x2E;` {
		t.Fatalf("hex: %q", got)
	}
	got = wp.KsesNormalizeEntities(`&amp;#x2E;`, "html")
	if got != `&amp;#x2E;` {
		t.Fatalf("double hex: %q", got)
	}
	// Invalid unicode codepoint stays disarmed.
	got = wp.KsesNormalizeEntities(`&#x110000;`, "html")
	if got != `&amp;#x110000;` {
		t.Fatalf("invalid unicode: %q", got)
	}
	// XML context: html-only named entities decode to characters.
	got = wp.KsesNormalizeEntities(`&copy; &amp;`, "xml")
	if !strings.Contains(got, "©") || !strings.Contains(got, "&amp;") {
		t.Fatalf("xml context: %q", got)
	}
}

func TestKsesUsesNormalizeEntitiesP2134(t *testing.T) {
	// Normalize runs before split; HTML-API-like split then unescapes valid named
	// entities in text, while invalid names stay disarmed as &amp;….
	out := wp.Kses(`<p>&copy; &notreal;</p>`, wp.AllowedHTMLPost(), nil)
	if !strings.Contains(out, `©`) {
		t.Fatalf("expected copyright decoded after normalize+split: %q", out)
	}
	if !strings.Contains(out, `&amp;notreal;`) {
		t.Fatalf("expected invalid named disarmed: %q", out)
	}
}

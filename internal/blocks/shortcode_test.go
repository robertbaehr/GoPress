package blocks_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/blocks"
)

func TestRenderShortcodeTextAttribute(t *testing.T) {
	raw := `<!-- wp:shortcode -->
[gallery ids="1,2"]
<!-- /wp:shortcode -->`
	out := blocks.Render(raw)
	if !strings.Contains(out, "[gallery ids=\"1,2\"]") {
		t.Fatalf("expected shortcode text preserved: %q", out)
	}
	if !strings.HasPrefix(strings.TrimSpace(out), "<p>") {
		t.Fatalf("expected wpautop wrap: %q", out)
	}
	if strings.Contains(out, "<!-- wp:") {
		t.Fatalf("leftover comments: %q", out)
	}
}

func TestRenderShortcodeTextJSONAttribute(t *testing.T) {
	raw := `<!-- wp:shortcode {"text":"[embed]https://example.com/v[/embed]"} /-->`
	out := blocks.Render(raw)
	if !strings.Contains(out, "[embed]https://example.com/v[/embed]") {
		t.Fatalf("text attr: %q", out)
	}
	if !strings.Contains(out, "<p>") {
		t.Fatalf("expected paragraph wrap: %q", out)
	}
}

func TestRenderShortcodeKeepsExistingParagraphs(t *testing.T) {
	raw := `<!-- wp:shortcode -->
<p>[caption]x[/caption]</p>
<!-- /wp:shortcode -->`
	out := blocks.Render(raw)
	if strings.Count(out, "<p>") != 1 {
		t.Fatalf("should not double-wrap: %q", out)
	}
}

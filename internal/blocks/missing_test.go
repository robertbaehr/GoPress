package blocks_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/blocks"
)

func TestRenderMissingOriginalContent(t *testing.T) {
	raw := `<!-- wp:missing {"originalName":"unregistered/example","originalContent":"<p>Recovered</p>"} /-->`
	out := blocks.Render(raw)
	if out != "<p>Recovered</p>" {
		t.Fatalf("missing originalContent: %q", out)
	}
}

func TestRenderMissingOriginalUndelimitedContent(t *testing.T) {
	raw := `<!-- wp:missing {"originalName":"old/widget","originalUndelimitedContent":"<div class=\"legacy\">Hi</div>"} /-->`
	out := blocks.Render(raw)
	if out != `<div class="legacy">Hi</div>` {
		t.Fatalf("missing originalUndelimitedContent: %q", out)
	}
}

func TestRenderMissingInnerHTMLFallback(t *testing.T) {
	raw := `<!-- wp:missing {"originalName":"x/y"} -->` + "\n" + `<p>Inner fallback</p>` + "\n" + `<!-- /wp:missing -->`
	out := blocks.Render(raw)
	if !strings.Contains(out, "Inner fallback") {
		t.Fatalf("missing inner HTML: %q", out)
	}
}

func TestRenderMissingPrefersOriginalContentOverInner(t *testing.T) {
	raw := `<!-- wp:missing {"originalName":"x/y","originalContent":"<p>From attr</p>"} -->` +
		`<p>Inner ignored</p><!-- /wp:missing -->`
	out := blocks.Render(raw)
	if out != "<p>From attr</p>" {
		t.Fatalf("prefer originalContent: %q", out)
	}
}

package blocks_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/blocks"
)

func TestMissingEmptyOriginalContentFallsThrough(t *testing.T) {
	raw := `<!-- wp:missing {"originalName":"x/y","originalContent":""} -->` +
		`<p class="inner">Body</p><!-- /wp:missing -->`
	out := blocks.Render(raw)
	if !strings.Contains(out, `class="inner"`) || !strings.Contains(out, "Body") {
		t.Fatalf("empty originalContent should fall through: %q", out)
	}
}

func TestMissingNullOriginalContentFallsThrough(t *testing.T) {
	raw := `<!-- wp:missing {"originalName":"x/y","originalContent":null} -->` +
		`<p>Null attr</p><!-- /wp:missing -->`
	out := blocks.Render(raw)
	if !strings.Contains(out, "Null attr") {
		t.Fatalf("null originalContent should fall through: %q", out)
	}
}

func TestMissingWhitespaceOriginalContentFallsThrough(t *testing.T) {
	raw := `<!-- wp:missing {"originalName":"x/y","originalContent":"   "} -->` +
		`<div>Whitespace skip</div><!-- /wp:missing -->`
	out := blocks.Render(raw)
	if !strings.Contains(out, "Whitespace skip") {
		t.Fatalf("whitespace originalContent should fall through: %q", out)
	}
}

func TestMissingInnerContentJoin(t *testing.T) {
	name := "core/missing"
	block := blocks.Block{
		BlockName:    &name,
		Attrs:        map[string]any{"originalName": "gone/block"},
		InnerContent: []any{"<p>", "Joined", "</p>"},
	}
	out := blocks.RenderBlock(block)
	if out != "<p>Joined</p>" {
		t.Fatalf("innerContent join: %q", out)
	}
}

func TestMissingSelfClosingOriginalContent(t *testing.T) {
	raw := `<!-- wp:missing {"originalName":"old/x","originalContent":"<div>Recovered self-close</div>"} /-->`
	out := blocks.Render(raw)
	if out != "<div>Recovered self-close</div>" {
		t.Fatalf("self-closing missing: %q", out)
	}
}

func TestMissingPrefersOriginalOverUndelimited(t *testing.T) {
	raw := `<!-- wp:missing {"originalName":"x/y","originalContent":"<p>OC</p>","originalUndelimitedContent":"<p>OUD</p>"} /-->`
	out := blocks.Render(raw)
	if out != "<p>OC</p>" {
		t.Fatalf("prefer originalContent: %q", out)
	}
}

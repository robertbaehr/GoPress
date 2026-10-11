package blocks_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/blocks"
)

// P5.146: core/quote honors fontSize/gradient; coerces align enum; drops center.
func TestRenderQuoteAttrsFontSizeGradientP5146(t *testing.T) {
	out := blocks.Render(`<!-- wp:quote {"align":"center","textAlign":"center","fontSize":"large","gradient":"vivid-cyan-blue-to-vivid-purple","className":"is-style-plain","anchor":"quote-wide","value":"<p>Hello quote</p>","citation":"Anon","style":{"spacing":{"padding":{"top":"8px"},"margin":{"bottom":"12px"}}}} /-->`)
	for _, want := range []string{
		`wp-block-quote`,
		`has-text-align-center`,
		`is-style-plain`,
		`has-large-font-size`,
		`has-background`,
		`has-vivid-cyan-blue-to-vivid-purple-gradient-background`,
		`id="quote-wide"`,
		`<p>Hello quote</p>`,
		`<cite>Anon</cite>`,
		`is-layout-flow`,
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in %q", want, out)
		}
	}
	if strings.Contains(out, "aligncenter") {
		t.Fatalf("unsupported align=center should be dropped: %q", out)
	}
}

func TestRenderQuoteAttrsAllowedAlignP5146(t *testing.T) {
	out := blocks.Render(`<!-- wp:quote {"align":"wide","value":"<p>Wide</p>"} /-->`)
	if !strings.Contains(out, "alignwide") {
		t.Fatalf("expected alignwide: %q", out)
	}
	out = blocks.Render(`<!-- wp:quote {"align":"full","value":"<p>Full</p>"} /-->`)
	if !strings.Contains(out, "alignfull") {
		t.Fatalf("expected alignfull: %q", out)
	}
}

func TestRenderQuoteAttrsSavedMarkupPolishP5146(t *testing.T) {
	saved := `<!-- wp:quote {"gradient":"luminous-vivid-amber-to-luminous-vivid-orange","fontSize":"small","textAlign":"right","anchor":"saved-quote"} --><blockquote class="wp-block-quote"><p>Saved quote</p><cite>Cite</cite></blockquote><!-- /wp:quote -->`
	out := blocks.Render(saved)
	for _, want := range []string{
		`wp-block-quote`,
		`has-text-align-right`,
		`has-small-font-size`,
		`has-luminous-vivid-amber-to-luminous-vivid-orange-gradient-background`,
		`id="saved-quote"`,
		`Saved quote`,
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in %q", want, out)
		}
	}
}

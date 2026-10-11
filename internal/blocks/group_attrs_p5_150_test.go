package blocks_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/blocks"
)

// P5.150: core/group honors fontSize/gradient/ariaLabel; coerces align enum; drops bogus.
func TestRenderGroupAttrsFontSizeGradientP5150(t *testing.T) {
	out := blocks.Render(`<!-- wp:group {"align":"bogus","fontSize":"large","gradient":"vivid-cyan-blue-to-vivid-purple","className":"is-style-default","anchor":"hero-group","ariaLabel":"Hero section","layout":{"type":"constrained"}} --><!-- wp:paragraph --><p>A</p><!-- /wp:paragraph --><!-- /wp:group -->`)
	for _, want := range []string{
		`wp-block-group`,
		`is-layout-constrained`,
		`is-style-default`,
		`has-large-font-size`,
		`has-background`,
		`has-vivid-cyan-blue-to-vivid-purple-gradient-background`,
		`id="hero-group"`,
		`aria-label="Hero section"`,
		`>A</p>`,
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in %q", want, out)
		}
	}
	if strings.Contains(out, "alignbogus") {
		t.Fatalf("unsupported align=bogus should be dropped: %q", out)
	}
}

func TestRenderGroupAttrsAllowedAlignP5150(t *testing.T) {
	for _, align := range []string{"wide", "full"} {
		out := blocks.Render(`<!-- wp:group {"align":"` + align + `","layout":{"type":"flow"}} --><!-- wp:paragraph --><p>X</p><!-- /wp:paragraph --><!-- /wp:group -->`)
		want := "align" + align
		if !strings.Contains(out, want) {
			t.Fatalf("expected %q in %q", want, out)
		}
	}
}

func TestRenderGroupAttrsSavedMarkupPolishP5150(t *testing.T) {
	saved := `<!-- wp:group {"gradient":"luminous-vivid-amber-to-luminous-vivid-orange","fontSize":"small","anchor":"saved-group","align":"wide","ariaLabel":"Saved group"} --><div class="wp-block-group"><p>Saved</p></div><!-- /wp:group -->`
	out := blocks.Render(saved)
	for _, want := range []string{
		`wp-block-group`,
		`alignwide`,
		`has-small-font-size`,
		`has-luminous-vivid-amber-to-luminous-vivid-orange-gradient-background`,
		`id="saved-group"`,
		`aria-label="Saved group"`,
		`Saved`,
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in %q", want, out)
		}
	}
}

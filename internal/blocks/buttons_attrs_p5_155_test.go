package blocks_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/blocks"
)

// P5.155: core/buttons honors fontSize/gradient; coerces align enum; drops bogus/none.
func TestRenderButtonsAttrsFontSizeGradientP5155(t *testing.T) {
	out := blocks.Render(`<!-- wp:buttons {"align":"bogus","fontSize":"large","gradient":"vivid-cyan-blue-to-vivid-purple","className":"is-style-default","anchor":"cta-row"} --><!-- wp:button {"text":"Go","url":"/go"} /--><!-- /wp:buttons -->`)
	for _, want := range []string{
		`wp-block-buttons`,
		`is-style-default`,
		`has-large-font-size`,
		`has-background`,
		`has-vivid-cyan-blue-to-vivid-purple-gradient-background`,
		`id="cta-row"`,
		`wp-block-button`,
		`/go`,
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in %q", want, out)
		}
	}
	if strings.Contains(out, "alignbogus") {
		t.Fatalf("unsupported align=bogus should be dropped: %q", out)
	}
}

func TestRenderButtonsAttrsAllowedAlignP5155(t *testing.T) {
	for _, align := range []string{"wide", "full"} {
		out := blocks.Render(`<!-- wp:buttons {"align":"` + align + `"} --><!-- wp:button {"text":"Go","url":"/go"} /--><!-- /wp:buttons -->`)
		want := "align" + align
		if !strings.Contains(out, want) {
			t.Fatalf("expected %q in %q", want, out)
		}
	}
	for _, bad := range []string{"none", "left", "center", "right"} {
		out := blocks.Render(`<!-- wp:buttons {"align":"` + bad + `"} --><!-- wp:button {"text":"Go","url":"/go"} /--><!-- /wp:buttons -->`)
		if strings.Contains(out, "align"+bad) {
			t.Fatalf("align=%s should be dropped: %q", bad, out)
		}
	}
}

func TestRenderButtonsAttrsSavedMarkupPolishP5155(t *testing.T) {
	saved := `<!-- wp:buttons {"fontSize":"small","gradient":"luminous-vivid-amber-to-luminous-vivid-orange","anchor":"saved-btns","align":"wide"} --><div class="wp-block-buttons is-layout-flex wp-block-buttons-is-layout-flex"><div class="wp-block-button"><a class="wp-block-button__link wp-element-button" href="/x">Saved</a></div></div><!-- /wp:buttons -->`
	out := blocks.Render(saved)
	for _, want := range []string{
		`wp-block-buttons`,
		`alignwide`,
		`has-small-font-size`,
		`has-luminous-vivid-amber-to-luminous-vivid-orange-gradient-background`,
		`id="saved-btns"`,
		`Saved`,
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in %q", want, out)
		}
	}
}

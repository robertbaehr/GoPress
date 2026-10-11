package blocks_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/blocks"
)

// P5.149: core/columns honors fontSize/gradient; coerces align enum; drops bogus.
func TestRenderColumnsAttrsFontSizeGradientP5149(t *testing.T) {
	out := blocks.Render(`<!-- wp:columns {"align":"bogus","fontSize":"large","gradient":"vivid-cyan-blue-to-vivid-purple","className":"is-style-default","anchor":"col-grid","isStackedOnMobile":false} --><!-- wp:column --><div class="wp-block-column"><p>A</p></div><!-- /wp:column --><!-- /wp:columns -->`)
	for _, want := range []string{
		`wp-block-columns`,
		`is-not-stacked-on-mobile`,
		`is-style-default`,
		`has-large-font-size`,
		`has-background`,
		`has-vivid-cyan-blue-to-vivid-purple-gradient-background`,
		`id="col-grid"`,
		`<p>A</p>`,
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in %q", want, out)
		}
	}
	if strings.Contains(out, "alignbogus") {
		t.Fatalf("unsupported align=bogus should be dropped: %q", out)
	}
}

func TestRenderColumnsAttrsAllowedAlignP5149(t *testing.T) {
	for _, align := range []string{"wide", "full"} {
		out := blocks.Render(`<!-- wp:columns {"align":"` + align + `"} --><!-- wp:column --><div class="wp-block-column"><p>X</p></div><!-- /wp:column --><!-- /wp:columns -->`)
		want := "align" + align
		if !strings.Contains(out, want) {
			t.Fatalf("expected %q in %q", want, out)
		}
	}
}

func TestRenderColumnsAttrsSavedMarkupPolishP5149(t *testing.T) {
	saved := `<!-- wp:columns {"gradient":"luminous-vivid-amber-to-luminous-vivid-orange","fontSize":"small","anchor":"saved-cols","align":"wide"} --><div class="wp-block-columns"><div class="wp-block-column"><p>Saved</p></div></div><!-- /wp:columns -->`
	out := blocks.Render(saved)
	for _, want := range []string{
		`wp-block-columns`,
		`alignwide`,
		`has-small-font-size`,
		`has-luminous-vivid-amber-to-luminous-vivid-orange-gradient-background`,
		`id="saved-cols"`,
		`Saved`,
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in %q", want, out)
		}
	}
}

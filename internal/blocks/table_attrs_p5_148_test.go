package blocks_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/blocks"
)

// P5.148: core/table honors fontSize/gradient; coerces align enum; drops bogus.
func TestRenderTableAttrsFontSizeGradientP5148(t *testing.T) {
	out := blocks.Render(`<!-- wp:table {"align":"bogus","fontSize":"large","gradient":"vivid-cyan-blue-to-vivid-purple","className":"is-style-stripes","anchor":"stats-table","hasFixedLayout":true,"body":[{"cells":[{"content":"A","tag":"td"},{"content":"B","tag":"td"}]}]} /-->`)
	for _, want := range []string{
		`wp-block-table`,
		`is-style-stripes`,
		`has-large-font-size`,
		`has-background`,
		`has-vivid-cyan-blue-to-vivid-purple-gradient-background`,
		`id="stats-table"`,
		`has-fixed-layout`,
		`<td>A</td>`,
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in %q", want, out)
		}
	}
	if strings.Contains(out, "alignbogus") {
		t.Fatalf("unsupported align=bogus should be dropped: %q", out)
	}
}

func TestRenderTableAttrsAllowedAlignP5148(t *testing.T) {
	for _, align := range []string{"left", "center", "right", "wide", "full"} {
		out := blocks.Render(`<!-- wp:table {"align":"` + align + `","body":[{"cells":[{"content":"X","tag":"td"}]}]} /-->`)
		want := "align" + align
		if !strings.Contains(out, want) {
			t.Fatalf("expected %q in %q", want, out)
		}
	}
}

func TestRenderTableAttrsSavedMarkupPolishP5148(t *testing.T) {
	saved := `<!-- wp:table {"gradient":"luminous-vivid-amber-to-luminous-vivid-orange","fontSize":"small","anchor":"saved-table","align":"wide"} --><figure class="wp-block-table"><table class="has-fixed-layout"><tbody><tr><td>Saved</td></tr></tbody></table></figure><!-- /wp:table -->`
	out := blocks.Render(saved)
	for _, want := range []string{
		`wp-block-table`,
		`alignwide`,
		`has-small-font-size`,
		`has-luminous-vivid-amber-to-luminous-vivid-orange-gradient-background`,
		`id="saved-table"`,
		`Saved`,
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in %q", want, out)
		}
	}
}

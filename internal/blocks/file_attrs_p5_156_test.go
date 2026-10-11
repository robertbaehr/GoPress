package blocks_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/blocks"
)

// P5.156: core/file honors fontSize/gradient; coerces align enum; drops bogus/none.
func TestRenderFileAttrsFontSizeGradientP5156(t *testing.T) {
	out := blocks.Render(`<!-- wp:file {"href":"https://example.com/doc.pdf","fileName":"Doc","align":"bogus","fontSize":"large","gradient":"vivid-cyan-blue-to-vivid-purple","className":"is-style-default","anchor":"file-doc"} /-->`)
	for _, want := range []string{
		`wp-block-file`,
		`is-style-default`,
		`has-large-font-size`,
		`has-background`,
		`has-vivid-cyan-blue-to-vivid-purple-gradient-background`,
		`id="file-doc"`,
		`https://example.com/doc.pdf`,
		`Doc`,
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in %q", want, out)
		}
	}
	if strings.Contains(out, "alignbogus") {
		t.Fatalf("unsupported align=bogus should be dropped: %q", out)
	}
}

func TestRenderFileAttrsAllowedAlignP5156(t *testing.T) {
	for _, align := range []string{"left", "center", "right", "wide", "full"} {
		out := blocks.Render(`<!-- wp:file {"href":"https://example.com/a.pdf","fileName":"A","align":"` + align + `"} /-->`)
		want := "align" + align
		if !strings.Contains(out, want) {
			t.Fatalf("expected %q in %q", want, out)
		}
	}
	none := blocks.Render(`<!-- wp:file {"href":"https://example.com/a.pdf","fileName":"A","align":"none"} /-->`)
	if strings.Contains(none, "alignnone") {
		t.Fatalf("default align=none should be dropped: %q", none)
	}
}

func TestRenderFileAttrsSavedMarkupPolishP5156(t *testing.T) {
	saved := `<!-- wp:file {"href":"https://example.com/bg.pdf","fileName":"Saved","fontSize":"small","gradient":"luminous-vivid-amber-to-luminous-vivid-orange","anchor":"saved-file","align":"wide"} --><div class="wp-block-file"><a href="https://example.com/bg.pdf">Saved</a></div><!-- /wp:file -->`
	out := blocks.Render(saved)
	for _, want := range []string{
		`wp-block-file`,
		`alignwide`,
		`has-small-font-size`,
		`has-luminous-vivid-amber-to-luminous-vivid-orange-gradient-background`,
		`id="saved-file"`,
		`Saved`,
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in %q", want, out)
		}
	}
}

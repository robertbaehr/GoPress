package blocks_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/blocks"
)

// P5.152: core/media-text honors fontSize/gradient; coerces align enum; drops bogus/none.
func TestRenderMediaTextAttrsFontSizeGradientP5152(t *testing.T) {
	out := blocks.Render(`<!-- wp:media-text {"mediaUrl":"https://example.com/hero.jpg","mediaType":"image","align":"bogus","fontSize":"large","gradient":"vivid-cyan-blue-to-vivid-purple","className":"is-style-default","anchor":"hero-mt","mediaAlt":"Hero media"} /-->`)
	for _, want := range []string{
		`wp-block-media-text`,
		`is-style-default`,
		`has-large-font-size`,
		`has-background`,
		`has-vivid-cyan-blue-to-vivid-purple-gradient-background`,
		`id="hero-mt"`,
		`alt="Hero media"`,
		`https://example.com/hero.jpg`,
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in %q", want, out)
		}
	}
	if strings.Contains(out, "alignbogus") {
		t.Fatalf("unsupported align=bogus should be dropped: %q", out)
	}
}

func TestRenderMediaTextAttrsAllowedAlignP5152(t *testing.T) {
	for _, align := range []string{"wide", "full"} {
		out := blocks.Render(`<!-- wp:media-text {"mediaUrl":"https://example.com/a.jpg","mediaType":"image","align":"` + align + `"} /-->`)
		want := "align" + align
		if !strings.Contains(out, want) {
			t.Fatalf("expected %q in %q", want, out)
		}
	}
	none := blocks.Render(`<!-- wp:media-text {"mediaUrl":"https://example.com/a.jpg","mediaType":"image","align":"none"} /-->`)
	if strings.Contains(none, "alignnone") {
		t.Fatalf("default align=none should be dropped: %q", none)
	}
}

func TestRenderMediaTextAttrsSavedMarkupPolishP5152(t *testing.T) {
	saved := `<!-- wp:media-text {"mediaUrl":"https://example.com/bg.jpg","fontSize":"small","gradient":"luminous-vivid-amber-to-luminous-vivid-orange","anchor":"saved-mt","align":"wide","mediaType":"image"} --><div class="wp-block-media-text is-stacked-on-mobile"><figure class="wp-block-media-text__media"><img src="https://example.com/bg.jpg" alt=""/></figure><div class="wp-block-media-text__content"><p>Saved</p></div></div><!-- /wp:media-text -->`
	out := blocks.Render(saved)
	for _, want := range []string{
		`wp-block-media-text`,
		`alignwide`,
		`has-small-font-size`,
		`has-luminous-vivid-amber-to-luminous-vivid-orange-gradient-background`,
		`id="saved-mt"`,
		`Saved`,
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in %q", want, out)
		}
	}
}

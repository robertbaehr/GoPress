package blocks_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/blocks"
)

// P5.151: core/cover honors fontSize/sizeSlug/alt; coerces align enum; drops bogus.
func TestRenderCoverAttrsFontSizeSizeSlugP5151(t *testing.T) {
	out := blocks.Render(`<!-- wp:cover {"url":"https://example.com/hero.jpg","align":"bogus","fontSize":"large","sizeSlug":"large","alt":"Hero cover","className":"is-style-default","anchor":"hero-cover","dimRatio":40} /-->`)
	for _, want := range []string{
		`wp-block-cover`,
		`is-style-default`,
		`has-large-font-size`,
		`size-large`,
		`alt="Hero cover"`,
		`id="hero-cover"`,
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

func TestRenderCoverAttrsAllowedAlignP5151(t *testing.T) {
	for _, align := range []string{"left", "center", "right", "wide", "full"} {
		out := blocks.Render(`<!-- wp:cover {"url":"https://example.com/a.jpg","align":"` + align + `","dimRatio":50} /-->`)
		want := "align" + align
		if !strings.Contains(out, want) {
			t.Fatalf("expected %q in %q", want, out)
		}
	}
}

func TestRenderCoverAttrsSavedMarkupPolishP5151(t *testing.T) {
	saved := `<!-- wp:cover {"url":"https://example.com/bg.jpg","fontSize":"small","anchor":"saved-cover","align":"wide","dimRatio":50} --><div class="wp-block-cover"><span aria-hidden="true" class="wp-block-cover__background has-background-dim"></span><div class="wp-block-cover__inner-container"><p>Saved</p></div></div><!-- /wp:cover -->`
	out := blocks.Render(saved)
	for _, want := range []string{
		`wp-block-cover`,
		`alignwide`,
		`has-small-font-size`,
		`id="saved-cover"`,
		`Saved`,
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in %q", want, out)
		}
	}
}

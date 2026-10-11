package blocks_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/blocks"
)

// P5.154: core/image honors aspectRatio/scale/title; coerces align enum; drops bogus/none.
func TestRenderImageAttrsAspectScaleTitleP5154(t *testing.T) {
	out := blocks.Render(`<!-- wp:image {"url":"https://example.com/hero.jpg","align":"bogus","aspectRatio":"16/9","scale":"cover","title":"Hero shot","className":"is-style-default","anchor":"hero-img","sizeSlug":"large","alt":"Hero"} /-->`)
	for _, want := range []string{
		`wp-block-image`,
		`is-style-default`,
		`size-large`,
		`aspect-ratio:16/9`,
		`object-fit:cover`,
		`title="Hero shot"`,
		`id="hero-img"`,
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

func TestRenderImageAttrsAllowedAlignP5154(t *testing.T) {
	for _, align := range []string{"left", "center", "right", "wide", "full"} {
		out := blocks.Render(`<!-- wp:image {"url":"https://example.com/a.jpg","align":"` + align + `"} /-->`)
		want := "align" + align
		if !strings.Contains(out, want) {
			t.Fatalf("expected %q in %q", want, out)
		}
	}
	none := blocks.Render(`<!-- wp:image {"url":"https://example.com/a.jpg","align":"none"} /-->`)
	if strings.Contains(none, "alignnone") {
		t.Fatalf("default align=none should be dropped: %q", none)
	}
}

func TestRenderImageAttrsSavedMarkupPolishP5154(t *testing.T) {
	saved := `<!-- wp:image {"aspectRatio":"4/3","scale":"contain","title":"Saved","anchor":"saved-img","align":"wide","isDecorative":true} --><figure class="wp-block-image"><img src="https://example.com/bg.jpg" alt="keep"/></figure><!-- /wp:image -->`
	out := blocks.Render(saved)
	for _, want := range []string{
		`wp-block-image`,
		`alignwide`,
		`aspect-ratio:4/3`,
		`object-fit:contain`,
		`role="presentation"`,
		`id="saved-img"`,
		`alt=""`,
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in %q", want, out)
		}
	}
	if strings.Contains(out, `title="Saved"`) {
		t.Fatalf("decorative image should drop title: %q", out)
	}
}

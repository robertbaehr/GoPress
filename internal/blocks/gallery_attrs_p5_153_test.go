package blocks_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/blocks"
)

// P5.153: core/gallery honors gradient; coerces align enum; drops bogus/none.
func TestRenderGalleryAttrsGradientP5153(t *testing.T) {
	out := blocks.Render(`<!-- wp:gallery {"ids":[],"align":"bogus","gradient":"vivid-cyan-blue-to-vivid-purple","className":"is-style-default","anchor":"hero-gal","columns":3,"imageCrop":true,"linkTo":"none"} --><figure class="wp-block-gallery has-nested-images columns-3 is-cropped"><figure class="wp-block-image size-large"><img src="https://example.com/a.jpg" alt=""/></figure></figure><!-- /wp:gallery -->`)
	for _, want := range []string{
		`wp-block-gallery`,
		`is-style-default`,
		`has-background`,
		`has-vivid-cyan-blue-to-vivid-purple-gradient-background`,
		`id="hero-gal"`,
		`https://example.com/a.jpg`,
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in %q", want, out)
		}
	}
	if strings.Contains(out, "alignbogus") {
		t.Fatalf("unsupported align=bogus should be dropped: %q", out)
	}
}

func TestRenderGalleryAttrsAllowedAlignP5153(t *testing.T) {
	for _, align := range []string{"left", "center", "right", "wide", "full"} {
		out := blocks.Render(`<!-- wp:gallery {"align":"` + align + `"} --><figure class="wp-block-gallery has-nested-images"><figure class="wp-block-image size-large"><img src="https://example.com/a.jpg" alt=""/></figure></figure><!-- /wp:gallery -->`)
		want := "align" + align
		if !strings.Contains(out, want) {
			t.Fatalf("expected %q in %q", want, out)
		}
	}
	none := blocks.Render(`<!-- wp:gallery {"align":"none"} --><figure class="wp-block-gallery has-nested-images"><figure class="wp-block-image size-large"><img src="https://example.com/a.jpg" alt=""/></figure></figure><!-- /wp:gallery -->`)
	if strings.Contains(none, "alignnone") {
		t.Fatalf("default align=none should be dropped: %q", none)
	}
}

func TestRenderGalleryAttrsSavedMarkupPolishP5153(t *testing.T) {
	saved := `<!-- wp:gallery {"gradient":"luminous-vivid-amber-to-luminous-vivid-orange","anchor":"saved-gal","align":"wide"} --><figure class="wp-block-gallery has-nested-images"><figure class="wp-block-image size-large"><img src="https://example.com/bg.jpg" alt=""/></figure></figure><!-- /wp:gallery -->`
	out := blocks.Render(saved)
	for _, want := range []string{
		`wp-block-gallery`,
		`alignwide`,
		`has-luminous-vivid-amber-to-luminous-vivid-orange-gradient-background`,
		`id="saved-gal"`,
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in %q", want, out)
		}
	}
}

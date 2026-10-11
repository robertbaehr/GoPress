package blocks_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/blocks"
)

// P5.158: core/video honors fontSize/gradient; coerces align enum; drops bogus/none.
func TestRenderVideoAttrsFontSizeGradientP5158(t *testing.T) {
	out := blocks.Render(`<!-- wp:video {"src":"https://example.com/a.mp4","align":"bogus","fontSize":"large","gradient":"vivid-cyan-blue-to-vivid-purple","className":"is-style-default","anchor":"video-clip"} /-->`)
	for _, want := range []string{
		`wp-block-video`,
		`is-style-default`,
		`has-large-font-size`,
		`has-background`,
		`has-vivid-cyan-blue-to-vivid-purple-gradient-background`,
		`id="video-clip"`,
		`https://example.com/a.mp4`,
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in %q", want, out)
		}
	}
	if strings.Contains(out, "alignbogus") {
		t.Fatalf("unsupported align=bogus should be dropped: %q", out)
	}
}

func TestRenderVideoAttrsAllowedAlignP5158(t *testing.T) {
	for _, align := range []string{"left", "center", "right", "wide", "full"} {
		out := blocks.Render(`<!-- wp:video {"src":"https://example.com/a.mp4","align":"` + align + `"} /-->`)
		want := "align" + align
		if !strings.Contains(out, want) {
			t.Fatalf("expected %q in %q", want, out)
		}
	}
	none := blocks.Render(`<!-- wp:video {"src":"https://example.com/a.mp4","align":"none"} /-->`)
	if strings.Contains(none, "alignnone") {
		t.Fatalf("default align=none should be dropped: %q", none)
	}
}

func TestRenderVideoAttrsSavedMarkupPolishP5158(t *testing.T) {
	saved := `<!-- wp:video {"src":"https://example.com/b.mp4","fontSize":"small","gradient":"luminous-vivid-amber-to-luminous-vivid-orange","anchor":"saved-video","align":"wide"} --><figure class="wp-block-video"><video controls src="https://example.com/b.mp4"></video></figure><!-- /wp:video -->`
	out := blocks.Render(saved)
	for _, want := range []string{
		`wp-block-video`,
		`alignwide`,
		`has-small-font-size`,
		`has-luminous-vivid-amber-to-luminous-vivid-orange-gradient-background`,
		`id="saved-video"`,
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in %q", want, out)
		}
	}
}

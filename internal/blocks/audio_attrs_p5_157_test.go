package blocks_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/blocks"
)

// P5.157: core/audio honors fontSize/gradient; coerces align enum; drops bogus/none.
func TestRenderAudioAttrsFontSizeGradientP5157(t *testing.T) {
	out := blocks.Render(`<!-- wp:audio {"src":"https://example.com/a.mp3","align":"bogus","fontSize":"large","gradient":"vivid-cyan-blue-to-vivid-purple","className":"is-style-default","anchor":"audio-clip"} /-->`)
	for _, want := range []string{
		`wp-block-audio`,
		`is-style-default`,
		`has-large-font-size`,
		`has-background`,
		`has-vivid-cyan-blue-to-vivid-purple-gradient-background`,
		`id="audio-clip"`,
		`https://example.com/a.mp3`,
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in %q", want, out)
		}
	}
	if strings.Contains(out, "alignbogus") {
		t.Fatalf("unsupported align=bogus should be dropped: %q", out)
	}
}

func TestRenderAudioAttrsAllowedAlignP5157(t *testing.T) {
	for _, align := range []string{"left", "center", "right", "wide", "full"} {
		out := blocks.Render(`<!-- wp:audio {"src":"https://example.com/a.mp3","align":"` + align + `"} /-->`)
		want := "align" + align
		if !strings.Contains(out, want) {
			t.Fatalf("expected %q in %q", want, out)
		}
	}
	none := blocks.Render(`<!-- wp:audio {"src":"https://example.com/a.mp3","align":"none"} /-->`)
	if strings.Contains(none, "alignnone") {
		t.Fatalf("default align=none should be dropped: %q", none)
	}
}

func TestRenderAudioAttrsSavedMarkupPolishP5157(t *testing.T) {
	saved := `<!-- wp:audio {"src":"https://example.com/b.mp3","fontSize":"small","gradient":"luminous-vivid-amber-to-luminous-vivid-orange","anchor":"saved-audio","align":"wide"} --><figure class="wp-block-audio"><audio controls src="https://example.com/b.mp3"></audio></figure><!-- /wp:audio -->`
	out := blocks.Render(saved)
	for _, want := range []string{
		`wp-block-audio`,
		`alignwide`,
		`has-small-font-size`,
		`has-luminous-vivid-amber-to-luminous-vivid-orange-gradient-background`,
		`id="saved-audio"`,
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in %q", want, out)
		}
	}
}

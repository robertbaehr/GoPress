package blocks_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/blocks"
)

func TestRenderSeparatorSynthesizeOpacityAndAlign(t *testing.T) {
	out := blocks.Render(`<!-- wp:separator {"opacity":"css","align":"wide","className":"is-style-wide"} /-->`)
	for _, want := range []string{
		`<hr class="`,
		`wp-block-separator`,
		`has-css-opacity`,
		`alignwide`,
		`is-style-wide`,
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in %q", want, out)
		}
	}
}

func TestRenderSeparatorDefaultAlphaOpacity(t *testing.T) {
	out := blocks.Render(`<!-- wp:separator /-->`)
	if !strings.Contains(out, `has-alpha-channel-opacity`) {
		t.Fatalf("expected default alpha opacity class: %q", out)
	}
	if strings.Contains(out, `has-css-opacity`) {
		t.Fatalf("unexpected css opacity: %q", out)
	}
}

func TestRenderSeparatorTagNameDiv(t *testing.T) {
	out := blocks.Render(`<!-- wp:separator {"tagName":"div","opacity":"alpha-channel"} /-->`)
	if !strings.Contains(out, `<div class="wp-block-separator`) {
		t.Fatalf("expected div tag: %q", out)
	}
}

func TestRenderSeparatorPrefersSavedMarkup(t *testing.T) {
	src := `<!-- wp:separator -->
<hr class="wp-block-separator has-alpha-channel-opacity"/>
<!-- /wp:separator -->`
	out := blocks.Render(src)
	if strings.Count(out, `wp-block-separator`) != 1 {
		t.Fatalf("expected single saved separator: %q", out)
	}
	if !strings.Contains(out, `has-alpha-channel-opacity`) {
		t.Fatalf("lost saved class: %q", out)
	}
}

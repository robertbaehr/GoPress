package blocks_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/blocks"
)

// P5.142: align=center, gradient preset, spacing.margin, invalid tagName→hr.
func TestRenderSeparatorAttrsP5142(t *testing.T) {
	out := blocks.Render(`<!-- wp:separator {"opacity":"css","align":"center","className":"is-style-dots","anchor":"sep-c","gradient":"vivid-cyan-blue-to-vivid-purple","tagName":"span","style":{"spacing":{"margin":{"top":"12px","bottom":"24px"}}}} /-->`)
	for _, want := range []string{
		`<hr`,
		`wp-block-separator`,
		`has-css-opacity`,
		`aligncenter`,
		`is-style-dots`,
		`id="sep-c"`,
		`has-background`,
		`has-vivid-cyan-blue-to-vivid-purple-gradient-background`,
		`margin-top:12px`,
		`margin-bottom:24px`,
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in: %q", want, out)
		}
	}
	if strings.Contains(out, `<span`) {
		t.Fatalf("invalid tagName should coerce to hr: %q", out)
	}
}

func TestRenderSeparatorGradientOnSavedP5142(t *testing.T) {
	saved := `<!-- wp:separator {"gradient":"luminous-vivid-amber-to-luminous-vivid-orange","align":"center","anchor":"saved-grad"} --><hr class="wp-block-separator has-alpha-channel-opacity"/><!-- /wp:separator -->`
	got := blocks.Render(saved)
	for _, want := range []string{
		`has-luminous-vivid-amber-to-luminous-vivid-orange-gradient-background`,
		`aligncenter`,
		`id="saved-grad"`,
		`has-background`,
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("saved polish missing %q in %q", want, got)
		}
	}
}

func TestRenderSeparatorUnknownOpacityDefaultsP5142(t *testing.T) {
	out := blocks.Render(`<!-- wp:separator {"opacity":"mystery"} /-->`)
	if !strings.Contains(out, `has-alpha-channel-opacity`) {
		t.Fatalf("unknown opacity should default to alpha-channel: %q", out)
	}
	if strings.Contains(out, `has-css-opacity`) {
		t.Fatalf("unexpected css opacity: %q", out)
	}
}

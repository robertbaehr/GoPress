package blocks_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/blocks"
)

// P5.161: core/social-links coerces align to left|center|right; drops wide/full/none/bogus.
func TestRenderSocialLinksAttrsAlignP5161(t *testing.T) {
	inner := `<!-- wp:social-link {"url":"https://wordpress.org","service":"wordpress"} /-->`
	for _, align := range []string{"left", "center", "right"} {
		out := blocks.Render(`<!-- wp:social-links {"align":"` + align + `"} -->` + inner + `<!-- /wp:social-links -->`)
		want := "align" + align
		if !strings.Contains(out, want) {
			t.Fatalf("expected %q in %q", want, out)
		}
	}
	for _, bad := range []string{"wide", "full", "none", "bogus"} {
		out := blocks.Render(`<!-- wp:social-links {"align":"` + bad + `"} -->` + inner + `<!-- /wp:social-links -->`)
		if strings.Contains(out, "align"+bad) {
			t.Fatalf("align=%s should be dropped: %q", bad, out)
		}
	}
}

// P5.161: gradient + anchor polish on social-links wrapper.
func TestRenderSocialLinksAttrsGradientAnchorP5161(t *testing.T) {
	out := blocks.Render(`<!-- wp:social-links {"gradient":"vivid-cyan-blue-to-vivid-purple","anchor":"social-row","className":"is-style-logos-only","align":"center"} -->` +
		`<!-- wp:social-link {"url":"https://wordpress.org","service":"wordpress"} /-->` +
		`<!-- /wp:social-links -->`)
	for _, want := range []string{
		`wp-block-social-links`,
		`aligncenter`,
		`is-style-logos-only`,
		`id="social-row"`,
		`has-background`,
		`has-vivid-cyan-blue-to-vivid-purple-gradient-background`,
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in %q", want, out)
		}
	}
}

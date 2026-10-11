package blocks_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/blocks"
)

// P5.144: core/preformatted honors fontSize/gradient; drops unsupported align.
func TestRenderPreformattedAttrsP5144(t *testing.T) {
	out := blocks.Render(`<!-- wp:preformatted {"align":"wide","fontSize":"large","gradient":"vivid-cyan-blue-to-vivid-purple","className":"is-style-default","anchor":"pre-wide","content":"line one\nline two","style":{"spacing":{"padding":{"top":"8px"},"margin":{"bottom":"12px"}}}} /-->`)
	for _, want := range []string{
		`wp-block-preformatted`,
		`has-large-font-size`,
		`has-background`,
		`has-vivid-cyan-blue-to-vivid-purple-gradient-background`,
		`is-style-default`,
		`id="pre-wide"`,
		`padding-top:8px`,
		`margin-bottom:12px`,
		`line one`,
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in: %q", want, out)
		}
	}
	if strings.Contains(out, `alignwide`) || strings.Contains(out, `alignfull`) {
		t.Fatalf("unsupported align should be dropped: %q", out)
	}
}

func TestRenderPreformattedGradientOnSavedP5144(t *testing.T) {
	saved := `<!-- wp:preformatted {"gradient":"luminous-vivid-amber-to-luminous-vivid-orange","fontSize":"small","anchor":"saved-pre"} --><pre class="wp-block-preformatted">saved pre</pre><!-- /wp:preformatted -->`
	got := blocks.Render(saved)
	for _, want := range []string{
		`has-luminous-vivid-amber-to-luminous-vivid-orange-gradient-background`,
		`has-small-font-size`,
		`id="saved-pre"`,
		`has-background`,
		`saved pre`,
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("saved polish missing %q in %q", want, got)
		}
	}
}

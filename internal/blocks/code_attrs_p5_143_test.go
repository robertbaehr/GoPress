package blocks_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/blocks"
)

// P5.143: core/code honors fontSize/gradient; coerces invalid align; keeps wide.
func TestRenderCodeAttrsP5143(t *testing.T) {
	out := blocks.Render(`<!-- wp:code {"align":"full","fontSize":"large","gradient":"vivid-cyan-blue-to-vivid-purple","className":"is-style-default","anchor":"snip-wide","content":"echo hi;","style":{"spacing":{"padding":{"top":"8px"},"margin":{"bottom":"12px"}}}} /-->`)
	for _, want := range []string{
		`wp-block-code`,
		`has-large-font-size`,
		`has-background`,
		`has-vivid-cyan-blue-to-vivid-purple-gradient-background`,
		`is-style-default`,
		`id="snip-wide"`,
		`padding-top:8px`,
		`margin-bottom:12px`,
		`echo hi;`,
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in: %q", want, out)
		}
	}
	if strings.Contains(out, `alignfull`) {
		t.Fatalf("invalid align=full should be coerced away: %q", out)
	}
}

func TestRenderCodeAlignWideKeptP5143(t *testing.T) {
	out := blocks.Render(`<!-- wp:code {"align":"wide","content":"ok();"} /-->`)
	if !strings.Contains(out, `alignwide`) {
		t.Fatalf("expected alignwide: %q", out)
	}
}

func TestRenderCodeGradientOnSavedP5143(t *testing.T) {
	saved := `<!-- wp:code {"gradient":"luminous-vivid-amber-to-luminous-vivid-orange","fontSize":"small","anchor":"saved-code"} --><pre class="wp-block-code"><code>saved();</code></pre><!-- /wp:code -->`
	got := blocks.Render(saved)
	for _, want := range []string{
		`has-luminous-vivid-amber-to-luminous-vivid-orange-gradient-background`,
		`has-small-font-size`,
		`id="saved-code"`,
		`has-background`,
		`saved();`,
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("saved polish missing %q in %q", want, got)
		}
	}
}

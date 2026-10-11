package blocks_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/blocks"
)

// P5.145: core/verse honors fontSize/gradient/textAlign; drops unsupported align.
func TestRenderVerseAttrsP5145(t *testing.T) {
	out := blocks.Render(`<!-- wp:verse {"align":"wide","textAlign":"center","fontSize":"large","gradient":"vivid-cyan-blue-to-vivid-purple","className":"is-style-default","anchor":"verse-wide","content":"line one\nline two","style":{"spacing":{"padding":{"top":"8px"},"margin":{"bottom":"12px"}}}} /-->`)
	for _, want := range []string{
		`wp-block-verse`,
		`has-text-align-center`,
		`has-large-font-size`,
		`has-background`,
		`has-vivid-cyan-blue-to-vivid-purple-gradient-background`,
		`is-style-default`,
		`id="verse-wide"`,
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

func TestRenderVerseGradientOnSavedP5145(t *testing.T) {
	saved := `<!-- wp:verse {"gradient":"luminous-vivid-amber-to-luminous-vivid-orange","fontSize":"small","textAlign":"right","anchor":"saved-verse"} --><pre class="wp-block-verse">saved verse</pre><!-- /wp:verse -->`
	got := blocks.Render(saved)
	for _, want := range []string{
		`has-luminous-vivid-amber-to-luminous-vivid-orange-gradient-background`,
		`has-small-font-size`,
		`has-text-align-right`,
		`id="saved-verse"`,
		`has-background`,
		`saved verse`,
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("saved polish missing %q in %q", want, got)
		}
	}
}

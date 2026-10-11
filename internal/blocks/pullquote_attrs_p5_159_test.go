package blocks_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/blocks"
)

// P5.159: core/pullquote honors fontSize/gradient; coerces align; drops center/bogus.
func TestRenderPullquoteAttrsFontSizeGradientP5159(t *testing.T) {
	out := blocks.Render(`<!-- wp:pullquote {"value":"Quoted","citation":"Anon","align":"center","fontSize":"large","gradient":"vivid-cyan-blue-to-vivid-purple","className":"is-style-solid-color","anchor":"pull-q"} /-->`)
	for _, want := range []string{
		`wp-block-pullquote`,
		`is-style-solid-color`,
		`has-large-font-size`,
		`has-background`,
		`has-vivid-cyan-blue-to-vivid-purple-gradient-background`,
		`id="pull-q"`,
		`Quoted`,
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in %q", want, out)
		}
	}
	if strings.Contains(out, "aligncenter") {
		t.Fatalf("align=center not in block.json enum: %q", out)
	}
}

func TestRenderPullquoteAttrsAllowedAlignP5159(t *testing.T) {
	for _, align := range []string{"left", "right", "wide", "full"} {
		out := blocks.Render(`<!-- wp:pullquote {"value":"Q","align":"` + align + `"} /-->`)
		want := "align" + align
		if !strings.Contains(out, want) {
			t.Fatalf("expected %q in %q", want, out)
		}
	}
	none := blocks.Render(`<!-- wp:pullquote {"value":"Q","align":"none"} /-->`)
	if strings.Contains(none, "alignnone") {
		t.Fatalf("align=none should be dropped: %q", none)
	}
}

func TestRenderPullquoteAttrsSavedMarkupPolishP5159(t *testing.T) {
	saved := `<!-- wp:pullquote {"fontSize":"small","gradient":"luminous-vivid-amber-to-luminous-vivid-orange","anchor":"saved-pq","align":"wide"} --><figure class="wp-block-pullquote"><blockquote><p>Saved</p></blockquote></figure><!-- /wp:pullquote -->`
	out := blocks.Render(saved)
	for _, want := range []string{
		`wp-block-pullquote`,
		`alignwide`,
		`has-small-font-size`,
		`has-luminous-vivid-amber-to-luminous-vivid-orange-gradient-background`,
		`id="saved-pq"`,
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in %q", want, out)
		}
	}
}

package blocks_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/blocks"
)

// P5.147: core/list honors fontSize/gradient; coerces align enum; drops left.
func TestRenderListAttrsFontSizeGradientP5147(t *testing.T) {
	out := blocks.Render(`<!-- wp:list {"align":"left","fontSize":"large","gradient":"vivid-cyan-blue-to-vivid-purple","className":"is-style-plain","anchor":"toc-list","values":"<li>One</li><li>Two</li>"} /-->`)
	for _, want := range []string{
		`wp-block-list`,
		`is-style-plain`,
		`has-large-font-size`,
		`has-background`,
		`has-vivid-cyan-blue-to-vivid-purple-gradient-background`,
		`id="toc-list"`,
		`<li>One</li>`,
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in %q", want, out)
		}
	}
	if strings.Contains(out, "alignleft") {
		t.Fatalf("unsupported align=left should be dropped: %q", out)
	}
}

func TestRenderListAttrsAllowedAlignP5147(t *testing.T) {
	out := blocks.Render(`<!-- wp:list {"align":"wide","values":"<li>Wide</li>"} /-->`)
	if !strings.Contains(out, "alignwide") {
		t.Fatalf("expected alignwide: %q", out)
	}
	out = blocks.Render(`<!-- wp:list {"align":"full","values":"<li>Full</li>"} /-->`)
	if !strings.Contains(out, "alignfull") {
		t.Fatalf("expected alignfull: %q", out)
	}
}

func TestRenderListAttrsSavedMarkupPolishP5147(t *testing.T) {
	saved := `<!-- wp:list {"ordered":true,"start":4,"reversed":true,"type":"i","gradient":"luminous-vivid-amber-to-luminous-vivid-orange","fontSize":"small","anchor":"saved-list"} --><ol><li>Saved item</li></ol><!-- /wp:list -->`
	out := blocks.Render(saved)
	for _, want := range []string{
		`wp-block-list`,
		`has-small-font-size`,
		`has-luminous-vivid-amber-to-luminous-vivid-orange-gradient-background`,
		`id="saved-list"`,
		`start="4"`,
		`reversed`,
		`type="i"`,
		`Saved item`,
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in %q", want, out)
		}
	}
}

package blocks_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/blocks"
)

func TestRenderGroupSynthesizeAndLayout(t *testing.T) {
	src := `<!-- wp:group {"layout":{"type":"flex","orientation":"vertical","justifyContent":"center"}} -->
<!-- wp:paragraph -->
<p>Inside</p>
<!-- /wp:paragraph -->
<!-- /wp:group -->`
	out := blocks.Render(src)
	for _, want := range []string{
		`wp-block-group`,
		`is-layout-flex`,
		`wp-block-group-is-layout-flex`,
		`is-vertical`,
		`is-content-justification-center`,
		`Inside`,
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in %q", want, out)
		}
	}
}

func TestRenderGroupTagName(t *testing.T) {
	src := `<!-- wp:group {"tagName":"section","layout":{"type":"constrained"}} -->
<!-- wp:paragraph --><p>Sec</p><!-- /wp:paragraph -->
<!-- /wp:group -->`
	out := blocks.Render(src)
	if !strings.Contains(out, "<section") || !strings.Contains(out, "</section>") {
		t.Fatalf("expected section tag: %q", out)
	}
	if !strings.Contains(out, "is-layout-constrained") || !strings.Contains(out, "has-global-padding") {
		t.Fatalf("expected constrained layout: %q", out)
	}
	if !strings.Contains(out, "Sec") {
		t.Fatalf("missing content: %q", out)
	}
}

func TestRenderGroupSavedFixture(t *testing.T) {
	out := blocks.Render(readSerialized(t, "core__group"))
	if !strings.Contains(out, `class="wp-block-group`) {
		t.Fatalf("missing group class: %q", truncate(out, 240))
	}
	if !strings.Contains(out, "This is a group block.") {
		t.Fatalf("missing content: %q", truncate(out, 240))
	}
	if strings.Contains(out, "<!-- wp:") {
		t.Fatalf("leftover comments: %q", truncate(out, 240))
	}
}

func TestRenderGroupDefaultFlowOnSynthesize(t *testing.T) {
	src := `<!-- wp:group -->
<!-- wp:paragraph --><p>Flow</p><!-- /wp:paragraph -->
<!-- /wp:group -->`
	out := blocks.Render(src)
	if !strings.Contains(out, "is-layout-flow") || !strings.Contains(out, "wp-block-group-is-layout-flow") {
		t.Fatalf("expected default flow layout: %q", out)
	}
}

func TestRenderGroupClassNameAnchorPolish(t *testing.T) {
	// P5.107: className/anchor on synthesize + saved markup.
	src := `<!-- wp:group {"anchor":"hero","className":"my-group","align":"wide","layout":{"type":"constrained"}} -->
<!-- wp:paragraph --><p>G</p><!-- /wp:paragraph -->
<!-- /wp:group -->`
	out := blocks.Render(src)
	for _, want := range []string{
		`id="hero"`,
		"my-group",
		"alignwide",
		"is-layout-constrained",
		"has-global-padding",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in %q", want, out)
		}
	}

	saved := `<!-- wp:group {"anchor":"saved-g","className":"extra"} -->
<div class="wp-block-group"><!-- wp:paragraph -->
<p>Saved</p>
<!-- /wp:paragraph --></div>
<!-- /wp:group -->`
	out = blocks.Render(saved)
	if !strings.Contains(out, `id="saved-g"`) || !strings.Contains(out, "extra") {
		t.Fatalf("saved polish: %q", out)
	}
}

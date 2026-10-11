package blocks_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/blocks"
)

func TestRenderHeadingSynthesizeLevelAlign(t *testing.T) {
	out := blocks.Render(`<!-- wp:heading {"level":3,"textAlign":"center","align":"wide","className":"hero","content":"Hello"} /-->`)
	for _, want := range []string{
		`<h3 class="`,
		`wp-block-heading`,
		`has-text-align-center`,
		`alignwide`,
		`hero`,
		`Hello`,
		`</h3>`,
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in %q", want, out)
		}
	}
}

func TestRenderHeadingPrefersSavedMarkupAddsClass(t *testing.T) {
	src := `<!-- wp:heading -->
<h2 class="alignwide">Saved</h2>
<!-- /wp:heading -->`
	out := blocks.Render(src)
	if !strings.Contains(out, `Saved`) {
		t.Fatalf("lost saved heading: %q", out)
	}
	if !strings.Contains(out, `wp-block-heading`) {
		t.Fatalf("expected wp-block-heading class: %q", out)
	}
	if !strings.Contains(out, `alignwide`) {
		t.Fatalf("lost align class: %q", out)
	}
}

func TestRenderParagraphSynthesizeDropCap(t *testing.T) {
	out := blocks.Render(`<!-- wp:paragraph {"dropCap":true,"textAlign":"left","className":"lede","content":"Once upon a time"} /-->`)
	for _, want := range []string{
		`<p class="`,
		`wp-block-paragraph`,
		`has-drop-cap`,
		`has-text-align-left`,
		`lede`,
		`Once upon a time`,
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in %q", want, out)
		}
	}
}

func TestRenderParagraphPrefersSavedMarkup(t *testing.T) {
	src := `<!-- wp:paragraph -->
<p>World</p>
<!-- /wp:paragraph -->`
	out := blocks.Render(src)
	if !strings.Contains(out, `World`) {
		t.Fatalf("lost saved paragraph: %q", out)
	}
	if !strings.Contains(out, `wp-block-paragraph`) {
		t.Fatalf("expected wp-block-paragraph: %q", out)
	}
}

func TestRenderHeadingAnchorClassNameOnSaved(t *testing.T) {
	src := `<!-- wp:heading {"level":2,"className":"is-style-default","anchor":"intro","textAlign":"right"} -->
<h2>Saved</h2>
<!-- /wp:heading -->`
	out := blocks.Render(src)
	for _, want := range []string{
		`wp-block-heading`,
		`is-style-default`,
		`has-text-align-right`,
		`id="intro"`,
		`Saved`,
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in %q", want, out)
		}
	}
}

func TestRenderHeadingSynthesizeAnchor(t *testing.T) {
	out := blocks.Render(`<!-- wp:heading {"level":4,"anchor":"sec-4","content":"Section"} /-->`)
	if !strings.Contains(out, `<h4`) || !strings.Contains(out, `id="sec-4"`) {
		t.Fatalf("expected h4 + anchor: %q", out)
	}
}

func TestRenderParagraphAnchorDropCapOnSaved(t *testing.T) {
	src := `<!-- wp:paragraph {"dropCap":true,"className":"lede","anchor":"lede-1","align":"wide"} -->
<p>Once</p>
<!-- /wp:paragraph -->`
	out := blocks.Render(src)
	for _, want := range []string{
		`wp-block-paragraph`,
		`has-drop-cap`,
		`lede`,
		`alignwide`,
		`id="lede-1"`,
		`Once`,
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in %q", want, out)
		}
	}
}

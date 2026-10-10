package blocks_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/blocks"
)

func TestRenderColumnsSynthesizeAndStackAttrs(t *testing.T) {
	src := `<!-- wp:columns {"verticalAlignment":"center","isStackedOnMobile":false} -->
<!-- wp:column {"width":"33.33%"} -->
<!-- wp:paragraph -->
<p>One</p>
<!-- /wp:paragraph -->
<!-- /wp:column -->
<!-- wp:column -->
<!-- wp:paragraph -->
<p>Two</p>
<!-- /wp:paragraph -->
<!-- /wp:column -->
<!-- /wp:columns -->`
	out := blocks.Render(src)
	if !strings.Contains(out, `class="wp-block-columns`) {
		t.Fatalf("missing columns wrapper: %q", out)
	}
	if !strings.Contains(out, "are-vertically-aligned-center") {
		t.Fatalf("missing vertical align class: %q", out)
	}
	if !strings.Contains(out, "is-not-stacked-on-mobile") {
		t.Fatalf("missing stack class: %q", out)
	}
	if !strings.Contains(out, "wp-block-column") || !strings.Contains(out, "One") || !strings.Contains(out, "Two") {
		t.Fatalf("missing columns/content: %q", out)
	}
	if !strings.Contains(out, "flex-basis:33.33%") {
		t.Fatalf("missing column width: %q", out)
	}
}

func TestRenderColumnsSavedFixture(t *testing.T) {
	out := blocks.Render(readSerialized(t, "core__columns"))
	if !strings.Contains(out, `class="wp-block-columns`) || !strings.Contains(out, "Column One, Paragraph One") {
		t.Fatalf("fixture render: %q", truncate(out, 240))
	}
	if strings.Contains(out, "<!-- wp:") {
		t.Fatalf("leftover comments: %q", truncate(out, 240))
	}
}

func TestRenderColumnsIsNotStackedFixture(t *testing.T) {
	out := blocks.Render(readSerialized(t, "core__columns__is-not-stacked-on-mobile"))
	if !strings.Contains(out, "is-not-stacked-on-mobile") {
		t.Fatalf("expected is-not-stacked-on-mobile: %q", truncate(out, 240))
	}
}

func TestRenderColumnVerticalAlign(t *testing.T) {
	src := `<!-- wp:column {"verticalAlignment":"bottom","width":"50%"} -->
<!-- wp:paragraph --><p>Hi</p><!-- /wp:paragraph -->
<!-- /wp:column -->`
	out := blocks.Render(src)
	if !strings.Contains(out, "is-vertically-aligned-bottom") {
		t.Fatalf("missing v-align: %q", out)
	}
	if !strings.Contains(out, "flex-basis:50%") {
		t.Fatalf("missing width: %q", out)
	}
}

func TestRenderColumnsPolishAnchorAlignColors(t *testing.T) {
	// P5.106: polish saved markup + applySupports colors.
	src := `<!-- wp:columns {"anchor":"hero-cols","align":"wide","className":"is-style-default","textColor":"contrast","backgroundColor":"base","isStackedOnMobile":false} -->
<div class="wp-block-columns"><!-- wp:column {"anchor":"col-a"} -->
<div class="wp-block-column"><!-- wp:paragraph -->
<p>A</p>
<!-- /wp:paragraph --></div>
<!-- /wp:column --></div>
<!-- /wp:columns -->`
	out := blocks.Render(src)
	for _, want := range []string{
		`wp-block-columns`,
		`alignwide`,
		`is-style-default`,
		`id="hero-cols"`,
		`id="col-a"`,
		`is-not-stacked-on-mobile`,
		`has-contrast-color`,
		`has-base-background-color`,
		`>A</p>`,
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in %q", want, out)
		}
	}
}

func TestRenderColumnsSynthesizeAnchorAlign(t *testing.T) {
	src := `<!-- wp:columns {"anchor":"grid","align":"full","className":"my-cols","verticalAlignment":"top","isStackedOnMobile":false} -->
<!-- wp:column {"anchor":"left","width":"40%","className":"col-left"} -->
<!-- wp:paragraph --><p>L</p><!-- /wp:paragraph -->
<!-- /wp:column -->
<!-- /wp:columns -->`
	out := blocks.Render(src)
	for _, want := range []string{
		`id="grid"`,
		`alignfull`,
		`my-cols`,
		`are-vertically-aligned-top`,
		`is-not-stacked-on-mobile`,
		`id="left"`,
		`col-left`,
		`flex-basis:40%`,
		`>L</p>`,
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in %q", want, out)
		}
	}
}

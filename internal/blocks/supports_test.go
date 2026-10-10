package blocks_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/blocks"
)

func TestBlockSupportsColorClasses(t *testing.T) {
	in := `<!-- wp:paragraph {"backgroundColor":"vivid-red","textColor":"white","fontSize":"large"} -->
<p>Hi</p>
<!-- /wp:paragraph -->`
	out := blocks.Render(in)
	for _, want := range []string{
		"has-background",
		"has-vivid-red-background-color",
		"has-text-color",
		"has-white-color",
		"has-large-font-size",
		"wp-block-paragraph",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in %q", want, out)
		}
	}
}

func TestBlockSupportsSpacingStyles(t *testing.T) {
	in := `<!-- wp:group {"style":{"spacing":{"padding":{"top":"var:preset|spacing|40","bottom":"20px"}},"color":{"background":"#abcdef"}}} -->
<div class="wp-block-group">X</div>
<!-- /wp:group -->`
	out := blocks.Render(in)
	if !strings.Contains(out, "has-background") {
		t.Fatalf("missing has-background: %q", out)
	}
	if !strings.Contains(out, "background-color:#abcdef") {
		t.Fatalf("missing bg style: %q", out)
	}
	if !strings.Contains(out, "padding-top:var(--wp--preset--spacing--40)") {
		t.Fatalf("missing padding preset: %q", out)
	}
	if !strings.Contains(out, "padding-bottom:20px") {
		t.Fatalf("missing padding bottom: %q", out)
	}
}

func TestBlockSupportsDuotonePreset(t *testing.T) {
	in := `<!-- wp:image {"style":{"color":{"duotone":"var:preset|duotone|blue-orange"}}} -->
<figure class="wp-block-image"><img src="/x.png" alt=""/></figure>
<!-- /wp:image -->`
	out := blocks.Render(in)
	if !strings.Contains(out, "wp-duotone-blue-orange") {
		t.Fatalf("missing duotone class: %q", out)
	}
	if !strings.Contains(out, "filter:var(--wp--preset--duotone--blue-orange)") {
		t.Fatalf("missing duotone filter style: %q", out)
	}
}

func TestBlockSupportsDuotoneFilterPath(t *testing.T) {
	in := `<!-- wp:image {"style":{"filter":{"duotone":"var(--wp--preset--duotone--grayscale)"}}} -->
<figure class="wp-block-image"><img src="/x.png" alt=""/></figure>
<!-- /wp:image -->`
	out := blocks.Render(in)
	if !strings.Contains(out, "wp-duotone-grayscale") {
		t.Fatalf("missing duotone class: %q", out)
	}
}

func TestBlockSupportsDuotoneCustomColors(t *testing.T) {
	in := `<!-- wp:image {"style":{"color":{"duotone":["#000000","#ffffff"]}}} -->
<figure class="wp-block-image"><img src="/x.png" alt=""/></figure>
<!-- /wp:image -->`
	out := blocks.Render(in)
	if !strings.Contains(out, "wp-duotone-000000-ffffff") {
		t.Fatalf("missing custom duotone class: %q", out)
	}
}

func TestBlockSupportsTypographyExtended(t *testing.T) {
	in := `<!-- wp:paragraph {"fontFamily":"manrope","style":{"typography":{"letterSpacing":"0.05em","fontStyle":"italic","textDecoration":"underline","textTransform":"uppercase","writingMode":"horizontal-tb","textColumns":"2"}}} -->
<p>Hi</p>
<!-- /wp:paragraph -->`
	out := blocks.Render(in)
	for _, want := range []string{
		"has-manrope-font-family",
		"letter-spacing:0.05em",
		"font-style:italic",
		"text-decoration:underline",
		"text-transform:uppercase",
		"writing-mode:horizontal-tb",
		"column-count:2",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in %q", want, out)
		}
	}
}

func TestBlockSupportsShadowPreset(t *testing.T) {
	in := `<!-- wp:group {"style":{"shadow":"var:preset|shadow|natural"}} -->
<div class="wp-block-group">X</div>
<!-- /wp:group -->`
	out := blocks.Render(in)
	if !strings.Contains(out, "box-shadow:var(--wp--preset--shadow--natural)") {
		t.Fatalf("missing shadow preset style: %q", out)
	}
}

func TestBlockSupportsShadowCustom(t *testing.T) {
	in := `<!-- wp:group {"style":{"shadow":"2px 2px 4px #000"}} -->
<div class="wp-block-group">X</div>
<!-- /wp:group -->`
	out := blocks.Render(in)
	if !strings.Contains(out, "box-shadow:2px 2px 4px #000") {
		t.Fatalf("missing custom shadow style: %q", out)
	}
}

func TestBlockSupportsShadowTopLevelAttr(t *testing.T) {
	in := `<!-- wp:group {"shadow":"var:preset|shadow|deep"} -->
<div class="wp-block-group">X</div>
<!-- /wp:group -->`
	out := blocks.Render(in)
	if !strings.Contains(out, "box-shadow:var(--wp--preset--shadow--deep)") {
		t.Fatalf("missing top-level shadow style: %q", out)
	}
}

func TestBlockSupportsPositionSticky(t *testing.T) {
	in := `<!-- wp:group {"style":{"position":{"type":"sticky","top":"0px"}}} -->
<div class="wp-block-group">X</div>
<!-- /wp:group -->`
	out := blocks.Render(in)
	if !strings.Contains(out, "is-position-sticky") {
		t.Fatalf("missing position class: %q", out)
	}
	if !strings.Contains(out, "position:sticky") {
		t.Fatalf("missing position style: %q", out)
	}
	if !strings.Contains(out, "z-index:10") {
		t.Fatalf("missing z-index: %q", out)
	}
	if !strings.Contains(out, "top:calc(0px + var(--wp-admin--admin-bar--position-offset, 0px))") {
		t.Fatalf("missing sticky top calc: %q", out)
	}
}

func TestBlockSupportsPositionFixed(t *testing.T) {
	in := `<!-- wp:group {"style":{"position":{"type":"fixed","top":"2rem","right":"1rem"}}} -->
<div class="wp-block-group">X</div>
<!-- /wp:group -->`
	out := blocks.Render(in)
	if !strings.Contains(out, "is-position-fixed") {
		t.Fatalf("missing fixed class: %q", out)
	}
	if !strings.Contains(out, "position:fixed") {
		t.Fatalf("missing fixed style: %q", out)
	}
	if !strings.Contains(out, "right:1rem") {
		t.Fatalf("missing right style: %q", out)
	}
}

func TestBlockSupportsLayoutFlexClasses(t *testing.T) {
	in := `<!-- wp:group {"layout":{"type":"flex","orientation":"vertical","justifyContent":"center","flexWrap":"nowrap","verticalAlignment":"top"}} -->
<div class="wp-block-group">X</div>
<!-- /wp:group -->`
	out := blocks.Render(in)
	for _, want := range []string{
		"is-layout-flex",
		"wp-block-group-is-layout-flex",
		"is-vertical",
		"is-content-justification-center",
		"is-nowrap",
		"display:flex",
		"flex-direction:column",
		"flex-wrap:nowrap",
		"align-items:center",
		"justify-content:flex-start",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in %q", want, out)
		}
	}
}

func TestBlockSupportsLayoutGridClasses(t *testing.T) {
	in := `<!-- wp:group {"layout":{"type":"grid","columnCount":3}} -->
<div class="wp-block-group">X</div>
<!-- /wp:group -->`
	out := blocks.Render(in)
	for _, want := range []string{
		"is-layout-grid",
		"wp-block-group-is-layout-grid",
		"display:grid",
		"grid-template-columns:repeat(3, minmax(0, 1fr))",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in %q", want, out)
		}
	}
}

func TestBlockSupportsLayoutFlexHorizontalJustify(t *testing.T) {
	in := `<!-- wp:group {"layout":{"type":"flex","justifyContent":"space-between","verticalAlignment":"center"}} -->
<div class="wp-block-group">X</div>
<!-- /wp:group -->`
	out := blocks.Render(in)
	for _, want := range []string{
		"is-layout-flex",
		"is-content-justification-space-between",
		"display:flex",
		"justify-content:space-between",
		"align-items:center",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in %q", want, out)
		}
	}
}

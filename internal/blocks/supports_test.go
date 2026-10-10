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

func TestBlockSupportsCSSFilterFunctions(t *testing.T) {
	// P5.35: safecss-allowed CSS filter functions via style.filter.{brightness,…}.
	in := `<!-- wp:image {"style":{"filter":{"brightness":"1.1","contrast":"0.9","blur":"2px"}}} -->
<figure class="wp-block-image"><img src="/x.png" alt=""/></figure>
<!-- /wp:image -->`
	out := blocks.Render(in)
	for _, want := range []string{
		"filter:",
		"blur(2px)",
		"brightness(1.1)",
		"contrast(0.9)",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in %q", want, out)
		}
	}
}

func TestBlockSupportsDuotoneComposesWithCSSFilters(t *testing.T) {
	in := `<!-- wp:image {"style":{"filter":{"duotone":"var:preset|duotone|grayscale","brightness":"1.05"}}} -->
<figure class="wp-block-image"><img src="/x.png" alt=""/></figure>
<!-- /wp:image -->`
	out := blocks.Render(in)
	if !strings.Contains(out, "wp-duotone-grayscale") {
		t.Fatalf("missing duotone class: %q", out)
	}
	if !strings.Contains(out, "var(--wp--preset--duotone--grayscale)") {
		t.Fatalf("missing duotone filter value: %q", out)
	}
	if !strings.Contains(out, "brightness(1.05)") {
		t.Fatalf("missing brightness filter: %q", out)
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

func TestBlockSupportsBorderSidesAndClasses(t *testing.T) {
	in := `<!-- wp:group {"borderColor":"vivid-cyan-blue","style":{"border":{"width":"2","radius":"8","top":{"color":"#111","width":"3px","style":"dashed"},"bottom":{"color":"var:preset|color|vivid-red","width":"1px","style":"solid"}}}} -->
<div class="wp-block-group">X</div>
<!-- /wp:group -->`
	out := blocks.Render(in)
	for _, want := range []string{
		"has-border-color",
		"has-vivid-cyan-blue-border-color",
		"border-color:var(--wp--preset--color--vivid-cyan-blue)",
		"border-width:2px",
		"border-radius:8px",
		"border-top-color:#111",
		"border-top-width:3px",
		"border-top-style:dashed",
		"border-bottom-color:var(--wp--preset--color--vivid-red)",
		"border-bottom-width:1px",
		"border-bottom-style:solid",
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

func TestBlockSupportsBackgroundImage(t *testing.T) {
	in := `<!-- wp:group {"style":{"background":{"backgroundImage":{"url":"https://example.com/bg.jpg"},"backgroundRepeat":"no-repeat"}}} -->
<div class="wp-block-group">X</div>
<!-- /wp:group -->`
	out := blocks.Render(in)
	for _, want := range []string{
		"has-background",
		`background-image:url('https://example.com/bg.jpg')`,
		"background-size:cover",
		"background-repeat:no-repeat",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in %q", want, out)
		}
	}
}

func TestBlockSupportsBackgroundContainCenters(t *testing.T) {
	in := `<!-- wp:cover {"style":{"background":{"backgroundImage":{"url":"/media/a.png"},"backgroundSize":"contain"}}} -->
<div class="wp-block-cover">X</div>
<!-- /wp:cover -->`
	out := blocks.Render(in)
	for _, want := range []string{
		"has-background",
		"background-size:contain",
		"background-position:50% 50%",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in %q", want, out)
		}
	}
}

func TestBlockSupportsBackgroundGradient(t *testing.T) {
	in := `<!-- wp:group {"style":{"background":{"gradient":"linear-gradient(red,blue)"}}} -->
<div class="wp-block-group">X</div>
<!-- /wp:group -->`
	out := blocks.Render(in)
	if !strings.Contains(out, "has-background") {
		t.Fatalf("missing has-background: %q", out)
	}
	if !strings.Contains(out, "background-image:linear-gradient(red,blue)") {
		t.Fatalf("missing gradient background-image: %q", out)
	}
}

func TestBlockSupportsDimensionsMinHeight(t *testing.T) {
	in := `<!-- wp:group {"style":{"dimensions":{"minHeight":"40vh","width":"100%"}}} -->
<div class="wp-block-group">X</div>
<!-- /wp:group -->`
	out := blocks.Render(in)
	for _, want := range []string{
		"min-height:40vh",
		"width:100%",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in %q", want, out)
		}
	}
}

func TestBlockSupportsDimensionsAspectRatio(t *testing.T) {
	in := `<!-- wp:group {"style":{"dimensions":{"aspectRatio":"16/9","minHeight":"40vh"}}} -->
<div class="wp-block-group">X</div>
<!-- /wp:group -->`
	out := blocks.Render(in)
	for _, want := range []string{
		"has-aspect-ratio",
		"aspect-ratio:16/9",
		"min-height:unset",
		"height:unset",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in %q", want, out)
		}
	}
}

func TestBlockSupportsTopLevelMinHeight(t *testing.T) {
	in := `<!-- wp:cover {"minHeight":"320px"} -->
<div class="wp-block-cover">X</div>
<!-- /wp:cover -->`
	out := blocks.Render(in)
	if !strings.Contains(out, "min-height:320px") {
		t.Fatalf("missing top-level minHeight: %q", out)
	}
}

func TestBlockSupportsElementsLinkAndButtonColors(t *testing.T) {
	in := `<!-- wp:group {"style":{"elements":{"link":{"color":{"text":"var:preset|color|luminous-vivid-amber"},":hover":{"color":{"text":"#ff0000"}}},"button":{"color":{"text":"#ffffff","background":"#111111"}}}}} -->
<div class="wp-block-group">X</div>
<!-- /wp:group -->`
	out := blocks.Render(in)
	for _, want := range []string{
		"wp-elements-",
		"has-link-color",
		"a:where(:not(.wp-element-button)){color:var(--wp--preset--color--luminous-vivid-amber)}",
		"a:where(:not(.wp-element-button)):hover{color:#ff0000}",
		".wp-element-button, .",
		"color:#ffffff",
		"background-color:#111111",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in %q", want, out)
		}
	}
}

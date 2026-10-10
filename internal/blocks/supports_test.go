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

package blocks_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/blocks"
)

func TestCoverSavedMarkup(t *testing.T) {
	src := `<!-- wp:cover {"url":"https://example.com/bg.jpg","dimRatio":50} -->
<div class="wp-block-cover"><span aria-hidden="true" class="wp-block-cover__background has-background-dim"></span><div class="wp-block-cover__inner-container"><p class="has-text-align-center">Guten Berg!</p></div></div>
<!-- /wp:cover -->`
	out := blocks.Render(src)
	if !strings.Contains(out, `class="wp-block-cover`) || !strings.Contains(out, "Guten Berg!") {
		t.Fatalf("expected saved cover markup: %q", out)
	}
}

func TestCoverSynthesizeFromURL(t *testing.T) {
	src := `<!-- wp:cover {"url":"https://example.com/hero.jpg","dimRatio":40} /-->`
	out := blocks.Render(src)
	if !strings.Contains(out, `class="wp-block-cover`) {
		t.Fatalf("missing cover class: %q", out)
	}
	if !strings.Contains(out, `wp-block-cover__image-background`) {
		t.Fatalf("missing image background: %q", out)
	}
	if !strings.Contains(out, `https://example.com/hero.jpg`) {
		t.Fatalf("missing url: %q", out)
	}
	if !strings.Contains(out, `wp-block-cover__inner-container`) {
		t.Fatalf("missing inner container: %q", out)
	}
}

func TestCoverSynthesizeFromAttachmentID(t *testing.T) {
	env := &blocks.Env{
		GetAttachmentURL: func(id int64) string {
			if id == 7 {
				return "https://cdn.example/a.jpg"
			}
			return ""
		},
	}
	out := env.Render(`<!-- wp:cover {"id":7,"dimRatio":50} /-->`)
	if !strings.Contains(out, "https://cdn.example/a.jpg") {
		t.Fatalf("expected attachment url: %q", out)
	}
}

func TestCoverFeaturedImageInjection(t *testing.T) {
	env := &blocks.Env{}
	env.SetCurrentPost(&blocks.QueryPost{
		ID:               1,
		FeaturedImageURL: "https://example.test/featured.jpg",
		FeaturedImageAlt: "Hero",
	})
	src := `<!-- wp:cover {"useFeaturedImage":true,"backgroundType":"image","focalPoint":{"x":0.25,"y":0.75}} -->
<div class="wp-block-cover"><span aria-hidden="true" class="wp-block-cover__background has-background-dim"></span><div class="wp-block-cover__inner-container"><p>Hi</p></div></div>
<!-- /wp:cover -->`
	out := env.Render(src)
	if !strings.Contains(out, `src="https://example.test/featured.jpg"`) {
		t.Fatalf("expected featured img: %q", out)
	}
	if !strings.Contains(out, `data-object-fit="cover"`) {
		t.Fatalf("expected object-fit: %q", out)
	}
	if !strings.Contains(out, `object-position:25% 75%`) {
		t.Fatalf("expected focal point: %q", out)
	}
	// Image must appear before inner container.
	img := strings.Index(out, `src="https://example.test/featured.jpg"`)
	inner := strings.Index(out, `wp-block-cover__inner-container`)
	if img < 0 || inner < 0 || img > inner {
		t.Fatalf("featured image should precede inner container: %q", out)
	}
}

func TestCoverFeaturedImageParallax(t *testing.T) {
	env := &blocks.Env{}
	env.SetCurrentPost(&blocks.QueryPost{
		ID:               2,
		FeaturedImageURL: "https://example.test/p.jpg",
	})
	src := `<!-- wp:cover {"useFeaturedImage":true,"hasParallax":true} -->
<div class="wp-block-cover has-parallax"><div class="wp-block-cover__inner-container"><p>X</p></div></div>
<!-- /wp:cover -->`
	out := env.Render(src)
	if !strings.Contains(out, `background-image:url(https://example.test/p.jpg)`) {
		t.Fatalf("expected parallax bg: %q", out)
	}
	if !strings.Contains(out, `has-parallax`) {
		t.Fatalf("expected has-parallax class: %q", out)
	}
}

func TestCoverEmpty(t *testing.T) {
	if out := blocks.Render(`<!-- wp:cover /-->`); strings.TrimSpace(out) != "" {
		t.Fatalf("empty cover should render empty: %q", out)
	}
}

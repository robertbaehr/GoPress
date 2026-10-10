package blocks_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/blocks"
)

func TestMediaTextSavedMarkup(t *testing.T) {
	src := `<!-- wp:media-text {"mediaType":"image"} -->
<div class="wp-block-media-text is-stacked-on-mobile"><figure class="wp-block-media-text__media"><img src="https://example.com/a.jpg" alt=""/></figure><div class="wp-block-media-text__content"><p>My Content</p></div></div>
<!-- /wp:media-text -->`
	out := blocks.Render(src)
	if !strings.Contains(out, `class="wp-block-media-text`) || !strings.Contains(out, "My Content") {
		t.Fatalf("expected saved media-text markup: %q", out)
	}
}

func TestMediaTextSynthesizeFromURL(t *testing.T) {
	src := `<!-- wp:media-text {"mediaUrl":"https://example.com/hero.jpg","mediaType":"image","mediaWidth":40} -->
<!-- wp:paragraph -->
<p>Side copy</p>
<!-- /wp:paragraph -->
<!-- /wp:media-text -->`
	out := blocks.Render(src)
	if !strings.Contains(out, `class="wp-block-media-text`) {
		t.Fatalf("missing media-text class: %q", out)
	}
	if !strings.Contains(out, `https://example.com/hero.jpg`) {
		t.Fatalf("missing media url: %q", out)
	}
	if !strings.Contains(out, `grid-template-columns:40% auto`) {
		t.Fatalf("missing media width style: %q", out)
	}
	if !strings.Contains(out, "Side copy") {
		t.Fatalf("missing inner content: %q", out)
	}
}

func TestMediaTextSynthesizeFromAttachmentID(t *testing.T) {
	env := &blocks.Env{
		GetAttachmentURL: func(id int64) string {
			if id == 9 {
				return "https://cdn.example/m.jpg"
			}
			return ""
		},
	}
	out := env.Render(`<!-- wp:media-text {"mediaId":9,"mediaType":"image"} /-->`)
	if !strings.Contains(out, "https://cdn.example/m.jpg") {
		t.Fatalf("expected attachment url: %q", out)
	}
	if !strings.Contains(out, "wp-image-9") {
		t.Fatalf("expected wp-image class: %q", out)
	}
}

func TestMediaTextMediaOnRight(t *testing.T) {
	src := `<!-- wp:media-text {"mediaUrl":"https://example.com/r.jpg","mediaPosition":"right","mediaType":"image"} -->
<!-- wp:paragraph --><p>Copy</p><!-- /wp:paragraph -->
<!-- /wp:media-text -->`
	out := blocks.Render(src)
	if !strings.Contains(out, "has-media-on-the-right") {
		t.Fatalf("expected right-media class: %q", out)
	}
	contentIdx := strings.Index(out, `wp-block-media-text__content`)
	mediaIdx := strings.Index(out, `wp-block-media-text__media`)
	if contentIdx < 0 || mediaIdx < 0 || contentIdx > mediaIdx {
		t.Fatalf("content should precede media when right: %q", out)
	}
}

func TestMediaTextFeaturedImageInjection(t *testing.T) {
	env := &blocks.Env{}
	env.SetCurrentPost(&blocks.QueryPost{
		ID:               1,
		FeaturedImageURL: "https://example.test/featured.jpg",
		FeaturedImageAlt: "Hero",
	})
	src := `<!-- wp:media-text {"useFeaturedImage":true,"imageFill":true,"focalPoint":{"x":0.25,"y":0.75}} -->
<div class="wp-block-media-text is-stacked-on-mobile is-image-fill"><figure class="wp-block-media-text__media"></figure><div class="wp-block-media-text__content"><p>Hi</p></div></div>
<!-- /wp:media-text -->`
	out := env.Render(src)
	if !strings.Contains(out, `src="https://example.test/featured.jpg"`) {
		t.Fatalf("expected featured img: %q", out)
	}
	if !strings.Contains(out, `wp-block-media-text__featured_image`) {
		t.Fatalf("expected featured image class: %q", out)
	}
	if !strings.Contains(out, `object-position:25% 75%`) {
		t.Fatalf("expected focal point: %q", out)
	}
	if !strings.Contains(out, "is-image-fill-element") {
		t.Fatalf("expected is-image-fill-element class: %q", out)
	}
	fig := strings.Index(out, `wp-block-media-text__media`)
	img := strings.Index(out, `src="https://example.test/featured.jpg"`)
	if fig < 0 || img < 0 || img < fig {
		t.Fatalf("featured image should be inside media figure: %q", out)
	}
}

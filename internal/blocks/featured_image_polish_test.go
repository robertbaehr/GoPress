package blocks_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/blocks"
)

func TestRenderPostFeaturedImagePolish(t *testing.T) {
	env := &blocks.Env{
		ListPosts: func(q blocks.QueryLoopArgs) []blocks.QueryPost {
			return []blocks.QueryPost{{
				ID: 7, Title: "With Thumb", Link: "/?p=7",
				FeaturedImageURL: "http://example.test/uploads/photo.jpg",
				FeaturedImageAlt: "A photo",
			}}
		},
	}
	in := `<!-- wp:query {"query":{"perPage":1}} -->
<div class="wp-block-query">
<!-- wp:post-template -->
<!-- wp:post-featured-image {"isLink":true,"linkTarget":"_blank","rel":"noopener","sizeSlug":"large","width":"320px","height":"200px","aspectRatio":"16/9","scale":"contain","align":"wide","dimRatio":50,"overlayColor":"black","className":"hero-thumb"} /-->
<!-- /wp:post-template -->
</div>
<!-- /wp:query -->`
	out := blocks.RenderWith(env, in)
	wants := []string{
		`class="wp-block-post-featured-image alignwide hero-thumb"`,
		`class="attachment-large size-large"`,
		`alt="With Thumb"`,
		`aspect-ratio:16/9`,
		`object-fit:contain`,
		`width:320px`,
		`height:200px`,
		`target="_blank"`,
		`rel="noopener"`,
		`href="/?p=7"`,
		`wp-block-post-featured-image__overlay`,
		`has-background-dim-50`,
		`has-black-background-color`,
	}
	for _, want := range wants {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in: %q", want, out)
		}
	}
}

func TestRenderPostFeaturedImageDefaultScaleNoLink(t *testing.T) {
	env := &blocks.Env{
		ListPosts: func(q blocks.QueryLoopArgs) []blocks.QueryPost {
			return []blocks.QueryPost{{
				ID: 1, Title: "Solo",
				FeaturedImageURL: "http://example.test/a.jpg",
				FeaturedImageAlt: "solo-alt",
			}}
		},
	}
	out := blocks.RenderWith(env, `<!-- wp:query {"query":{"perPage":1}} -->
<!-- wp:post-template -->
<!-- wp:post-featured-image /-->
<!-- /wp:post-template -->
<!-- /wp:query -->`)
	if !strings.Contains(out, `alt="solo-alt"`) {
		t.Fatalf("expected meta alt when not linked: %q", out)
	}
	if !strings.Contains(out, `object-fit:cover`) {
		t.Fatalf("expected default scale cover: %q", out)
	}
	if strings.Contains(out, `<a href=`) {
		t.Fatalf("isLink default false should not wrap: %q", out)
	}
	if strings.Contains(out, `__overlay`) {
		t.Fatalf("dimRatio 0 should omit overlay: %q", out)
	}
}

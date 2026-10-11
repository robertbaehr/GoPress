package blocks_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/blocks"
)

// P5.136: border attrs + shadow on <img>, overlay inherits border, supports.anchor on figure.
func TestRenderPostFeaturedImageBorderShadowAnchor(t *testing.T) {
	env := &blocks.Env{
		ListPosts: func(q blocks.QueryLoopArgs) []blocks.QueryPost {
			return []blocks.QueryPost{{
				ID: 9, Title: "Bordered", Link: "/?p=9",
				FeaturedImageURL: "http://example.test/uploads/hero.jpg",
				FeaturedImageAlt: "hero",
			}}
		},
	}
	in := `<!-- wp:query {"query":{"perPage":1}} -->
<!-- wp:post-template -->
<!-- wp:post-featured-image {"anchor":"feat-hero","borderColor":"vivid-cyan-blue","dimRatio":40,"overlayColor":"black","style":{"border":{"width":"2px","radius":"8px"},"shadow":"0 4px 8px rgba(0,0,0,0.2)"}} /-->
<!-- /wp:post-template -->
<!-- /wp:query -->`
	out := blocks.RenderWith(env, in)
	wants := []string{
		`id="feat-hero"`,
		`has-border-color`,
		`has-vivid-cyan-blue-border-color`,
		`border-width:2px`,
		`border-radius:8px`,
		`box-shadow:0 4px 8px rgba(0,0,0,0.2)`,
		`wp-block-post-featured-image__overlay`,
		`has-background-dim-40`,
	}
	for _, want := range wants {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in: %q", want, out)
		}
	}
	// Border class/style appear on both img and overlay (WP).
	imgIdx := strings.Index(out, `<img`)
	overlayIdx := strings.Index(out, `wp-block-post-featured-image__overlay`)
	if imgIdx < 0 || overlayIdx < 0 {
		t.Fatalf("missing img or overlay: %q", out)
	}
	imgSlice := out[imgIdx:overlayIdx]
	if !strings.Contains(imgSlice, "has-border-color") || !strings.Contains(imgSlice, "border-radius:8px") {
		t.Fatalf("expected border on img before overlay: %q", imgSlice)
	}
	overlaySlice := out[overlayIdx:]
	if !strings.Contains(overlaySlice, "has-border-color") {
		t.Fatalf("expected border class on overlay: %q", overlaySlice)
	}
}

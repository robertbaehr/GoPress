package blocks_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/blocks"
)

func TestRenderPostExcerptPolish(t *testing.T) {
	env := &blocks.Env{
		ListPosts: func(q blocks.QueryLoopArgs) []blocks.QueryPost {
			return []blocks.QueryPost{{
				ID: 4, Title: "Excerpted", Link: "/?p=4",
				Excerpt: "One two three four five six seven eight nine ten",
			}}
		},
	}
	out := blocks.RenderWith(env, `<!-- wp:query {"query":{"perPage":1}} -->
<!-- wp:post-template -->
<!-- wp:post-excerpt {"excerptLength":5,"moreText":"Continue","showMoreOnNewLine":true,"textAlign":"center","className":"teaser","style":{"elements":{"link":{"color":{"text":"red"}}}}} /-->
<!-- /wp:post-template -->
<!-- /wp:query -->`)
	wants := []string{
		`wp-block-post-excerpt has-text-align-center has-link-color teaser`,
		`wp-block-post-excerpt__excerpt`,
		`One two three four five…`,
		`wp-block-post-excerpt__more-text`,
		`wp-block-post-excerpt__more-link`,
		`href="/?p=4"`,
		`Continue`,
	}
	for _, want := range wants {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in: %q", want, out)
		}
	}
}

func TestRenderPostExcerptMoreInline(t *testing.T) {
	env := &blocks.Env{
		ListPosts: func(q blocks.QueryLoopArgs) []blocks.QueryPost {
			return []blocks.QueryPost{{
				ID: 5, Title: "Inline", Link: "/?p=5",
				Excerpt: "Short bit",
			}}
		},
	}
	out := blocks.RenderWith(env, `<!-- wp:query {"query":{"perPage":1}} -->
<!-- wp:post-template -->
<!-- wp:post-excerpt {"moreText":"More","showMoreOnNewLine":false} /-->
<!-- /wp:post-template -->
<!-- /wp:query -->`)
	if strings.Contains(out, "wp-block-post-excerpt__more-text") {
		t.Fatalf("inline more should not use more-text paragraph: %q", out)
	}
	if !strings.Contains(out, `Short bit <a class="wp-block-post-excerpt__more-link"`) &&
		!strings.Contains(out, `Short bit<a class="wp-block-post-excerpt__more-link"`) {
		// WP inserts a space separator when excerpt is non-empty.
		if !strings.Contains(out, "Short bit") || !strings.Contains(out, "more-link") {
			t.Fatalf("expected inline more link: %q", out)
		}
	}
}

func TestRenderPostExcerptFromContent(t *testing.T) {
	env := &blocks.Env{
		ListPosts: func(q blocks.QueryLoopArgs) []blocks.QueryPost {
			return []blocks.QueryPost{{
				ID: 6, Title: "Auto", Link: "/?p=6",
				Content: "<p>Alpha beta gamma delta epsilon zeta eta theta</p>",
			}}
		},
	}
	out := blocks.RenderWith(env, `<!-- wp:query {"query":{"perPage":1}} -->
<!-- wp:post-template -->
<!-- wp:post-excerpt {"excerptLength":3} /-->
<!-- /wp:post-template -->
<!-- /wp:query -->`)
	if !strings.Contains(out, "Alpha beta gamma…") {
		t.Fatalf("expected trimmed auto excerpt: %q", out)
	}
	if !strings.Contains(out, `class="wp-block-post-excerpt"`) {
		t.Fatalf("expected wrapper class: %q", out)
	}
}

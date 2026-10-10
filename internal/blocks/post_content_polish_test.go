package blocks_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/blocks"
)

func TestRenderPostTitlePolish(t *testing.T) {
	env := &blocks.Env{
		ListPosts: func(q blocks.QueryLoopArgs) []blocks.QueryPost {
			return []blocks.QueryPost{{
				ID: 3, Title: "Linked Title", Link: "/?p=3",
			}}
		},
	}
	in := `<!-- wp:query {"query":{"perPage":1}} -->
<!-- wp:post-template -->
<!-- wp:post-title {"isLink":true,"level":0,"textAlign":"center","linkTarget":"_blank","rel":"noopener","className":"hero-title","style":{"elements":{"link":{"color":{"text":"var:preset|color|accent-1"}}}}} /-->
<!-- /wp:post-template -->
<!-- /wp:query -->`
	out := blocks.RenderWith(env, in)
	wants := []string{
		`<p class="wp-block-post-title has-text-align-center has-link-color hero-title`,
		`href="/?p=3"`,
		`target="_blank"`,
		`rel="noopener"`,
		`Linked Title`,
	}
	for _, want := range wants {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in: %q", want, out)
		}
	}
}

func TestRenderPostTitleEmptyOmitsMarkup(t *testing.T) {
	env := &blocks.Env{
		ListPosts: func(q blocks.QueryLoopArgs) []blocks.QueryPost {
			return []blocks.QueryPost{{ID: 1, Title: "  ", Link: "/?p=1"}}
		},
	}
	out := blocks.RenderWith(env, `<!-- wp:query {"query":{"perPage":1}} -->
<!-- wp:post-template --><!-- wp:post-title /--><!-- /wp:post-template -->
<!-- /wp:query -->`)
	if strings.Contains(out, "wp-block-post-title") {
		t.Fatalf("empty title should omit markup: %q", out)
	}
}

func TestRenderPostContentTagNameAndClass(t *testing.T) {
	env := &blocks.Env{
		ListPosts: func(q blocks.QueryLoopArgs) []blocks.QueryPost {
			return []blocks.QueryPost{{
				ID: 9, Title: "T",
				Content: `<!-- wp:paragraph --><p>Hello</p><!-- /wp:paragraph -->`,
			}}
		},
	}
	out := blocks.RenderWith(env, `<!-- wp:query {"query":{"perPage":1}} -->
<!-- wp:post-template -->
<!-- wp:post-content {"tagName":"article","align":"wide","className":"entry-body"} /-->
<!-- /wp:post-template -->
<!-- /wp:query -->`)
	if !strings.Contains(out, `<article class="entry-content wp-block-post-content alignwide entry-body`) {
		t.Fatalf("expected article wrapper: %q", out)
	}
	if !strings.Contains(out, "Hello") {
		t.Fatalf("expected body: %q", out)
	}
}

func TestRenderPostContentReentrancyHalt(t *testing.T) {
	nested := `<!-- wp:post-content /-->`
	env := &blocks.Env{
		DebugDisplay: true,
		ListPosts: func(q blocks.QueryLoopArgs) []blocks.QueryPost {
			return []blocks.QueryPost{{
				ID: 4, Title: "Loop",
				Content: nested,
			}}
		},
	}
	out := blocks.RenderWith(env, `<!-- wp:query {"query":{"perPage":1}} -->
<!-- wp:post-template --><!-- wp:post-content /--><!-- /wp:post-template -->
<!-- /wp:query -->`)
	if !strings.Contains(out, "[block rendering halted]") {
		t.Fatalf("expected debug halt message: %q", out)
	}

	env.DebugDisplay = false
	outQuiet := blocks.RenderWith(env, `<!-- wp:query {"query":{"perPage":1}} -->
<!-- wp:post-template --><!-- wp:post-content /--><!-- /wp:post-template -->
<!-- /wp:query -->`)
	if strings.Contains(outQuiet, "[block rendering halted]") {
		t.Fatalf("quiet mode should hide halt: %q", outQuiet)
	}
}

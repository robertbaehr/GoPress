package blocks_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/blocks"
)

// P5.137: query-no-results wrapper honors align, fontSize, and anchor.
func TestRenderQueryNoResultsAlignFontSizeAnchorP5137(t *testing.T) {
	env := &blocks.Env{
		ListPosts: func(q blocks.QueryLoopArgs) []blocks.QueryPost { return nil },
	}
	in := `<!-- wp:query {"query":{"perPage":1}} -->
<div class="wp-block-query">
<!-- wp:post-template --><!-- wp:post-title /--><!-- /wp:post-template -->
<!-- wp:query-no-results {"align":"wide","fontSize":"medium","anchor":"empty-query","className":"is-style-default","backgroundColor":"base"} --><!-- wp:paragraph --><p>No matches</p><!-- /wp:paragraph --><!-- /wp:query-no-results -->
</div>
<!-- /wp:query -->`
	out := blocks.RenderWith(env, in)
	for _, want := range []string{
		`id="empty-query"`,
		"wp-block-query-no-results",
		"alignwide",
		"has-medium-font-size",
		"is-style-default",
		"has-base-background-color",
		"No matches",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in %q", want, out)
		}
	}
}

// P5.137: when the query has posts, no-results (and its attrs) stay hidden.
func TestRenderQueryNoResultsHiddenWhenPostsP5137(t *testing.T) {
	env := &blocks.Env{
		ListPosts: func(q blocks.QueryLoopArgs) []blocks.QueryPost {
			return []blocks.QueryPost{{ID: 1, Title: "Hi", Content: "x"}}
		},
	}
	in := `<!-- wp:query {"query":{"perPage":1}} -->
<div class="wp-block-query">
<!-- wp:post-template --><!-- wp:post-title /--><!-- /wp:post-template -->
<!-- wp:query-no-results {"anchor":"should-not-appear"} --><!-- wp:paragraph --><p>Empty</p><!-- /wp:paragraph --><!-- /wp:query-no-results -->
</div>
<!-- /wp:query -->`
	out := blocks.RenderWith(env, in)
	if strings.Contains(out, "should-not-appear") || strings.Contains(out, "Empty") {
		t.Fatalf("expected no-results hidden when posts exist: %q", out)
	}
}

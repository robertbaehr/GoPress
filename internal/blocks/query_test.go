package blocks_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/blocks"
)

func TestRenderQueryPostTemplate(t *testing.T) {
	env := &blocks.Env{
		ListPosts: func(q blocks.QueryLoopArgs) []blocks.QueryPost {
			if q.PerPage != 3 {
				t.Fatalf("perPage=%d", q.PerPage)
			}
			return []blocks.QueryPost{
				{ID: 1, Title: "Hello World", Content: "<!-- wp:paragraph --><p>Body</p><!-- /wp:paragraph -->", Date: "2024-01-02 03:04:05", Link: "/?p=1"},
				{ID: 2, Title: "Second", Content: "<p>More</p>", Date: "2024-01-03 00:00:00", Link: "/?p=2"},
			}
		},
	}
	in := `<!-- wp:query {"query":{"perPage":3,"postType":"post","inherit":true}} -->
<div class="wp-block-query">
<!-- wp:post-template -->
<!-- wp:post-title {"isLink":true} /-->
<!-- wp:post-date /-->
<!-- wp:post-content /-->
<!-- /wp:post-template -->
</div>
<!-- /wp:query -->`
	out := blocks.RenderWith(env, in)
	if !strings.Contains(out, `class="wp-block-post-template"`) {
		t.Fatalf("missing post-template: %q", out)
	}
	if !strings.Contains(out, `Hello World`) || !strings.Contains(out, `Second`) {
		t.Fatalf("missing titles: %q", out)
	}
	if !strings.Contains(out, `<a href="/?p=1"`) {
		t.Fatalf("missing title link: %q", out)
	}
	if !strings.Contains(out, `2024-01-02`) {
		t.Fatalf("missing date: %q", out)
	}
	if !strings.Contains(out, `wp-block-post-content`) {
		t.Fatalf("missing content: %q", out)
	}
	if strings.Contains(out, "<!-- wp:") {
		t.Fatalf("leftover comments: %q", out)
	}
}

func TestRenderPatternExpandsQuery(t *testing.T) {
	env := &blocks.Env{
		LoadPattern: func(slug string) (string, bool) {
			if slug != "demo/loop" {
				return "", false
			}
			return `<!-- wp:query {"query":{"perPage":1}} --><div class="wp-block-query"><!-- wp:post-template --><!-- wp:post-title /--><!-- /wp:post-template --></div><!-- /wp:query -->`, true
		},
		ListPosts: func(q blocks.QueryLoopArgs) []blocks.QueryPost {
			return []blocks.QueryPost{{ID: 9, Title: "From Pattern", Link: "/?p=9"}}
		},
	}
	out := blocks.RenderWith(env, `<!-- wp:pattern {"slug":"demo/loop"} /-->`)
	if !strings.Contains(out, "From Pattern") {
		t.Fatalf("pattern query missing post: %q", out)
	}
}

func TestRenderQueryNoResults(t *testing.T) {
	env := &blocks.Env{
		ListPosts: func(q blocks.QueryLoopArgs) []blocks.QueryPost { return nil },
	}
	in := `<!-- wp:query {"query":{"perPage":1}} -->
<div class="wp-block-query">
<!-- wp:post-template --><!-- wp:post-title /--><!-- /wp:post-template -->
<!-- wp:query-no-results --><!-- wp:paragraph --><p>Nothing here</p><!-- /wp:paragraph --><!-- /wp:query-no-results -->
</div>
<!-- /wp:query -->`
	out := blocks.RenderWith(env, in)
	if !strings.Contains(out, "Nothing here") {
		t.Fatalf("expected no-results: %q", out)
	}
}

package blocks_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/blocks"
)

func TestRenderPostMetaBlocks(t *testing.T) {
	env := &blocks.Env{
		HomeURL: func() string { return "http://example.test" },
		AuthorName: func(postID int64) string {
			if postID != 1 {
				t.Fatalf("postID=%d", postID)
			}
			return "Admin"
		},
		PostTerms: func(postID int64, taxonomy string) []blocks.NavLink {
			if postID != 1 || taxonomy != "category" {
				t.Fatalf("post=%d tax=%q", postID, taxonomy)
			}
			return []blocks.NavLink{
				{Label: "News", URL: "http://example.test/category/news/"},
				{Label: "Go", URL: "http://example.test/category/go/"},
			}
		},
		ListPosts: func(q blocks.QueryLoopArgs) []blocks.QueryPost {
			return []blocks.QueryPost{{
				ID:      1,
				Title:   "Hello Meta",
				Content: `<!-- wp:paragraph --><p>Body text here</p><!-- /wp:paragraph -->`,
				Excerpt: "Short excerpt",
				Date:    "2024-06-15 12:00:00",
				Link:    "http://example.test/?p=1",
			}}
		},
	}
	in := `<!-- wp:query {"query":{"perPage":1}} -->
<div class="wp-block-query">
<!-- wp:post-template -->
<!-- wp:post-title {"isLink":true,"level":3} /-->
<!-- wp:post-date {"isLink":true} /-->
<!-- wp:post-author {"isLink":true,"byline":"Written by"} /-->
<!-- wp:post-author-name /-->
<!-- wp:post-terms {"term":"category","separator":" · "} /-->
<!-- wp:post-excerpt /-->
<!-- wp:post-content /-->
<!-- /wp:post-template -->
</div>
<!-- /wp:query -->`
	out := blocks.RenderWith(env, in)
	checks := []string{
		`<h3 class="wp-block-post-title">`,
		`Hello Meta`,
		`wp-block-post-date`,
		`June 15, 2024`,
		`wp-block-post-author`,
		`wp-block-post-author__byline`,
		`Written by`,
		`wp-block-post-author__name`,
		`wp-block-post-author-name`,
		`taxonomy-category wp-block-post-terms`,
		`>News</a>`,
		`>Go</a>`,
		`wp-block-post-excerpt__excerpt`,
		`Short excerpt`,
		`entry-content wp-block-post-content`,
		`Body text here`,
	}
	for _, want := range checks {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in %q", want, out)
		}
	}
	if strings.Contains(out, "<!-- wp:") {
		t.Fatalf("leftover comments: %q", out)
	}
}

func TestRenderPostAuthorEmptyWithoutCallback(t *testing.T) {
	env := &blocks.Env{
		ListPosts: func(q blocks.QueryLoopArgs) []blocks.QueryPost {
			return []blocks.QueryPost{{ID: 2, Title: "T", Link: "/?p=2"}}
		},
	}
	out := blocks.RenderWith(env, `<!-- wp:query {"query":{"perPage":1}} --><div class="wp-block-query"><!-- wp:post-template --><!-- wp:post-author /--><!-- /wp:post-template --></div><!-- /wp:query -->`)
	if strings.Contains(out, "wp-block-post-author") {
		t.Fatalf("expected empty author without callback: %q", out)
	}
}

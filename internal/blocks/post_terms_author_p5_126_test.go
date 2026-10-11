package blocks_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/blocks"
)

func TestPostTermsAnchorColorsAndDefaultSeparator(t *testing.T) {
	env := &blocks.Env{
		HomeURL: func() string { return "http://example.test" },
		PostTerms: func(postID int64, taxonomy string) []blocks.NavLink {
			return []blocks.NavLink{
				{Label: "News", URL: "http://example.test/category/news/"},
				{Label: "Go", URL: "http://example.test/category/go/"},
			}
		},
		ListPosts: func(q blocks.QueryLoopArgs) []blocks.QueryPost {
			return []blocks.QueryPost{{ID: 1, Title: "T", Link: "/?p=1"}}
		},
	}
	out := blocks.RenderWith(env, `<!-- wp:query {"query":{"perPage":1}} -->
<div class="wp-block-query">
<!-- wp:post-template -->
<!-- wp:post-terms {"term":"category","anchor":"terms-top","textColor":"contrast","backgroundColor":"base","className":"my-terms","prefix":"In: ","suffix":"."} /-->
<!-- /wp:post-template -->
</div>
<!-- /wp:query -->`)
	for _, want := range []string{
		`id="terms-top"`,
		`taxonomy-category wp-block-post-terms`,
		`has-text-color`,
		`has-contrast-color`,
		`has-background`,
		`has-base-background-color`,
		`my-terms`,
		`wp-block-post-terms__prefix`,
		`wp-block-post-terms__suffix`,
		// block.json default separator ", "
		`wp-block-post-terms__separator">, </span>`,
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in %q", want, out)
		}
	}
}

func TestPostAuthorAvatarDefaultAnchorAndColors(t *testing.T) {
	env := &blocks.Env{
		HomeURL: func() string { return "http://example.test" },
		AuthorName: func(postID int64) string {
			return "Admin"
		},
		AuthorEmail: func(postID int64) string {
			return "admin@example.test"
		},
		AuthorLink: func(postID int64) string {
			return "http://example.test/author/admin/"
		},
		ListPosts: func(q blocks.QueryLoopArgs) []blocks.QueryPost {
			return []blocks.QueryPost{{ID: 1, Title: "T", Link: "/?p=1"}}
		},
	}
	// Omit showAvatar → block.json default true.
	out := blocks.RenderWith(env, `<!-- wp:query {"query":{"perPage":1}} -->
<div class="wp-block-query">
<!-- wp:post-template -->
<!-- wp:post-author {"anchor":"author-box","textColor":"accent","backgroundColor":"cyan-blue","isLink":true,"linkTarget":"_blank","className":"by-line"} /-->
<!-- /wp:post-template -->
</div>
<!-- /wp:query -->`)
	for _, want := range []string{
		`id="author-box"`,
		`wp-block-post-author__avatar`,
		`avatar avatar-48 photo`,
		`has-text-color`,
		`has-accent-color`,
		`has-background`,
		`has-cyan-blue-background-color`,
		`by-line`,
		`rel="noreferrer noopener"`,
		`target="_blank"`,
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in %q", want, out)
		}
	}

	// Explicit showAvatar:false suppresses avatar.
	outOff := blocks.RenderWith(env, `<!-- wp:query {"query":{"perPage":1}} -->
<div class="wp-block-query">
<!-- wp:post-template -->
<!-- wp:post-author {"showAvatar":false} /-->
<!-- /wp:post-template -->
</div>
<!-- /wp:query -->`)
	if strings.Contains(outOff, `wp-block-post-author__avatar`) {
		t.Fatalf("showAvatar:false should omit avatar: %q", outOff)
	}
}

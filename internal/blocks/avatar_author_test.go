package blocks_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/blocks"
)

func TestRenderAvatarAndAuthorBiography(t *testing.T) {
	env := &blocks.Env{
		HomeURL: func() string { return "http://example.test" },
		AuthorName: func(postID int64) string {
			if postID != 1 {
				t.Fatalf("postID=%d", postID)
			}
			return "Admin"
		},
		AuthorEmail: func(postID int64) string {
			return "admin@example.com"
		},
		AuthorBio: func(postID int64) string {
			return "Fixture administrator."
		},
		AuthorLink: func(postID int64) string {
			return "http://example.test/author/admin/"
		},
		ListPosts: func(q blocks.QueryLoopArgs) []blocks.QueryPost {
			return []blocks.QueryPost{{
				ID:    1,
				Title: "Hello",
				Link:  "http://example.test/?p=1",
			}}
		},
	}
	in := `<!-- wp:query {"query":{"perPage":1}} -->
<div class="wp-block-query">
<!-- wp:post-template -->
<!-- wp:avatar {"size":48,"isLink":true} /-->
<!-- wp:post-author-name {"isLink":true,"textAlign":"center"} /-->
<!-- wp:post-author-biography {"textAlign":"left"} /-->
<!-- /wp:post-template -->
</div>
<!-- /wp:query -->`
	out := blocks.RenderWith(env, in)
	checks := []string{
		`wp-block-avatar`,
		`wp-block-avatar__image`,
		`wp-block-avatar__link`,
		`avatar-48`,
		`href="http://example.test/author/admin/"`,
		`wp-block-post-author-name has-text-align-center`,
		`wp-block-post-author-name__link`,
		`wp-block-post-author-biography has-text-align-left`,
		`Fixture administrator.`,
	}
	for _, want := range checks {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in %q", want, out)
		}
	}
}

func TestRenderAvatarEmptyWithoutEmail(t *testing.T) {
	env := &blocks.Env{
		AuthorName: func(postID int64) string { return "Admin" },
		ListPosts: func(q blocks.QueryLoopArgs) []blocks.QueryPost {
			return []blocks.QueryPost{{ID: 1, Title: "T", Link: "/?p=1"}}
		},
	}
	out := blocks.RenderWith(env, `<!-- wp:query {"query":{"perPage":1}} --><div class="wp-block-query"><!-- wp:post-template --><!-- wp:avatar /--><!-- /wp:post-template --></div><!-- /wp:query -->`)
	if strings.Contains(out, "wp-block-avatar") {
		t.Fatalf("expected empty avatar without email: %q", out)
	}
}

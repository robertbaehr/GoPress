package blocks_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/blocks"
)

func TestRenderAvatarAndAuthorNamePolish(t *testing.T) {
	env := &blocks.Env{
		HomeURL:     func() string { return "http://example.test" },
		AuthorName:  func(int64) string { return "Ada" },
		AuthorEmail: func(int64) string { return "ada@example.com" },
		AuthorLink:  func(int64) string { return "http://example.test/author/ada/" },
		ListPosts: func(q blocks.QueryLoopArgs) []blocks.QueryPost {
			return []blocks.QueryPost{{ID: 1, Title: "Hello", Link: "/?p=1"}}
		},
	}
	out := blocks.RenderWith(env, `<!-- wp:query {"query":{"perPage":1}} -->
<!-- wp:post-template -->
<!-- wp:avatar {"size":64,"isLink":true,"linkTarget":"_blank","textAlign":"center","className":"byline-avatar","style":{"elements":{"link":{"color":{"text":"red"}}}}} /-->
<!-- wp:post-author-name {"isLink":true,"textAlign":"right","className":"byline-name","style":{"elements":{"link":{"color":{"text":"blue"}}}}} /-->
<!-- /wp:post-template -->
<!-- /wp:query -->`)
	wants := []string{
		`wp-block-avatar has-text-align-center has-link-color byline-avatar`,
		`avatar-64`,
		`wp-block-avatar__link`,
		`target="_blank"`,
		`aria-label="(Ada author archive, opens in a new tab)"`,
		`wp-block-post-author-name has-text-align-right has-link-color byline-name`,
		`wp-block-post-author-name__link`,
		`href="http://example.test/author/ada/"`,
	}
	for _, want := range wants {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in: %q", want, out)
		}
	}
}

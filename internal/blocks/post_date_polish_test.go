package blocks_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/blocks"
)

func TestRenderPostDatePolish(t *testing.T) {
	env := &blocks.Env{
		ListPosts: func(q blocks.QueryLoopArgs) []blocks.QueryPost {
			return []blocks.QueryPost{{
				ID: 3, Title: "Dated", Link: "/?p=3",
				Date: "2024-01-02 03:04:05", Modified: "2024-06-15 12:00:00",
			}}
		},
	}
	out := blocks.RenderWith(env, `<!-- wp:query {"query":{"perPage":1}} -->
<!-- wp:post-template -->
<!-- wp:post-date {"displayType":"modified","format":"Y-m-d","textAlign":"right","isLink":true,"linkTarget":"_blank","rel":"noopener","className":"meta-date","style":{"elements":{"link":{"color":{"text":"var:preset|color|accent-1"}}}}} /-->
<!-- /wp:post-template -->
<!-- /wp:query -->`)
	wants := []string{
		`wp-block-post-date wp-block-post-date__modified-date has-text-align-right has-link-color meta-date`,
		`2024-06-15`,
		`href="/?p=3"`,
		`target="_blank"`,
		`rel="noopener"`,
	}
	for _, want := range wants {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in: %q", want, out)
		}
	}
}

func TestRenderPostDateHumanDiff(t *testing.T) {
	env := &blocks.Env{
		ListPosts: func(q blocks.QueryLoopArgs) []blocks.QueryPost {
			return []blocks.QueryPost{{
				ID: 1, Title: "Old", Link: "/?p=1",
				Date: "2020-01-01 00:00:00",
			}}
		},
	}
	out := blocks.RenderWith(env, `<!-- wp:query {"query":{"perPage":1}} -->
<!-- wp:post-template --><!-- wp:post-date {"format":"human-diff"} /--><!-- /wp:post-template -->
<!-- /wp:query -->`)
	if !strings.Contains(out, " ago") {
		t.Fatalf("expected human-diff ago: %q", out)
	}
}

func TestRenderPostAuthorPolish(t *testing.T) {
	env := &blocks.Env{
		ListPosts: func(q blocks.QueryLoopArgs) []blocks.QueryPost {
			return []blocks.QueryPost{{ID: 2, Title: "By", Link: "/?p=2"}}
		},
		AuthorName:  func(int64) string { return "Ada" },
		AuthorEmail: func(int64) string { return "ada@example.com" },
		AuthorBio:   func(int64) string { return "Writes code." },
		AuthorLink:  func(int64) string { return "/author/ada/" },
	}
	out := blocks.RenderWith(env, `<!-- wp:query {"query":{"perPage":1}} -->
<!-- wp:post-template -->
<!-- wp:post-author {"showAvatar":true,"avatarSize":48,"showBio":true,"byline":"By","itemsJustification":"center","textAlign":"left","isLink":true,"className":"byline-box","style":{"elements":{"link":{"color":{"text":"red"}}}}} /-->
<!-- /wp:post-template -->
<!-- /wp:query -->`)
	wants := []string{
		`wp-block-post-author items-justified-center has-text-align-left has-link-color byline-box`,
		`wp-block-post-author__avatar`,
		`wp-block-post-author__bio`,
		`Writes code.`,
		`href="/author/ada/"`,
		`wp-block-post-author__byline`,
	}
	for _, want := range wants {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in: %q", want, out)
		}
	}
}

func TestRenderPostTermsPolish(t *testing.T) {
	env := &blocks.Env{
		ListPosts: func(q blocks.QueryLoopArgs) []blocks.QueryPost {
			return []blocks.QueryPost{{ID: 5, Title: "Tagged", Link: "/?p=5"}}
		},
		PostTerms: func(postID int64, tax string) []blocks.NavLink {
			if tax != "post_tag" {
				return nil
			}
			return []blocks.NavLink{
				{Label: "Go", URL: "/tag/go/"},
				{Label: "PHP", URL: "/tag/php/"},
			}
		},
	}
	out := blocks.RenderWith(env, `<!-- wp:query {"query":{"perPage":1}} -->
<!-- wp:post-template -->
<!-- wp:post-terms {"term":"post_tag","separator":" · ","prefix":"Tagged: ","suffix":".","textAlign":"center","className":"term-row","style":{"elements":{"link":{"color":{"text":"blue"}}}}} /-->
<!-- /wp:post-template -->
<!-- /wp:query -->`)
	wants := []string{
		`taxonomy-post_tag wp-block-post-terms has-text-align-center has-link-color term-row`,
		`wp-block-post-terms__prefix`,
		`Tagged: `,
		`wp-block-post-terms__separator`,
		` · `,
		`wp-block-post-terms__suffix`,
		`/tag/go/`,
		`/tag/php/`,
	}
	for _, want := range wants {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in: %q", want, out)
		}
	}
}

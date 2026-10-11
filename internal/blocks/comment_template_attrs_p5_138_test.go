package blocks_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/blocks"
)

// P5.138: comment-template ol wrapper honors align, fontSize, and anchor.
func TestRenderCommentTemplateAttrsP5138(t *testing.T) {
	env := &blocks.Env{
		ListComments: func(postID int64) []blocks.CommentItem {
			return []blocks.CommentItem{
				{ID: 1, Author: "Alice", Content: "Hi", Date: "2024-01-01 10:00:00"},
			}
		},
	}
	env.SetCurrentPost(&blocks.QueryPost{ID: 9, Title: "Post", Link: "/?p=9"})
	out := blocks.RenderWith(env, `<!-- wp:comment-template {"align":"wide","fontSize":"medium","anchor":"thread-root","className":"is-style-default"} /-->`)
	for _, want := range []string{
		`wp-block-comment-template`,
		`alignwide`,
		`has-medium-font-size`,
		`is-style-default`,
		`id="thread-root"`,
		`id="comment-1"`,
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in: %q", want, out)
		}
	}
}

func TestRenderCommentTemplateAttrsEmptyWithoutCommentsP5138(t *testing.T) {
	env := &blocks.Env{
		ListComments: func(postID int64) []blocks.CommentItem { return nil },
	}
	env.SetCurrentPost(&blocks.QueryPost{ID: 9, Title: "Post", Link: "/?p=9"})
	out := blocks.RenderWith(env, `<!-- wp:comment-template {"anchor":"should-not-appear"} /-->`)
	if strings.TrimSpace(out) != "" {
		t.Fatalf("expected empty without comments, got %q", out)
	}
}

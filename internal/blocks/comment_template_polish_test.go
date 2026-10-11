package blocks_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/blocks"
)

func TestRenderCommentsWrapperPolish(t *testing.T) {
	env := &blocks.Env{}
	env.SetCurrentPost(&blocks.QueryPost{ID: 1, Title: "Hello", Link: "/?p=1"})
	out := blocks.RenderWith(env, `<!-- wp:comments {"textAlign":"center","className":"thread","legacy":true} -->
<!-- wp:comments-title /-->
<!-- /wp:comments -->`)
	for _, want := range []string{
		`wp-block-comments`,
		`wp-block-post-comments`,
		`has-text-align-center`,
		`thread`,
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in: %q", want, out)
		}
	}
}

func TestRenderCommentsEmptyWithoutPost(t *testing.T) {
	out := blocks.RenderWith(&blocks.Env{}, `<!-- wp:comments --><!-- wp:comments-title /--><!-- /wp:comments -->`)
	if strings.TrimSpace(out) != "" {
		t.Fatalf("expected empty without post context, got %q", out)
	}
}

func TestRenderCommentTemplateOrderAndClasses(t *testing.T) {
	env := &blocks.Env{
		CommentOrder: "desc",
		ListComments: func(postID int64) []blocks.CommentItem {
			return []blocks.CommentItem{
				{ID: 1, Author: "Alice", AuthorEmail: "a@example.com", Content: "First", Date: "2024-01-01 10:00:00"},
				{ID: 2, Author: "Bob", AuthorEmail: "b@example.com", Content: "Second", Date: "2024-01-02 10:00:00"},
				{ID: 3, Parent: 1, Author: "Carol", Content: "Reply", Date: "2024-01-03 10:00:00"},
			}
		},
	}
	env.SetCurrentPost(&blocks.QueryPost{ID: 9, Title: "Post", Link: "/?p=9"})
	out := blocks.RenderWith(env, `<!-- wp:comment-template {"className":"nested"} /-->`)
	for _, want := range []string{
		`wp-block-comment-template nested`,
		`id="comment-2"`,
		`id="comment-1"`,
		`class="comment even thread-even depth-1"`,
		`class="comment odd thread-odd depth-1"`,
		`<ol>`,
		`depth-2`,
		`Alice`,
		`Bob`,
		`Carol`,
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in: %q", want, out)
		}
	}
	// desc: Bob (id 2) should appear before Alice (id 1) at top level.
	i2 := strings.Index(out, `id="comment-2"`)
	i1 := strings.Index(out, `id="comment-1"`)
	if i2 < 0 || i1 < 0 || i2 > i1 {
		t.Fatalf("expected desc order Bob before Alice: %q", out)
	}
}

func TestRenderCommentsTitleShowPostTitle(t *testing.T) {
	env := &blocks.Env{
		ListComments: func(postID int64) []blocks.CommentItem {
			return []blocks.CommentItem{{ID: 1, Author: "A", Content: "Hi"}}
		},
	}
	env.SetCurrentPost(&blocks.QueryPost{ID: 3, Title: "World", Link: "/?p=3"})
	out := blocks.RenderWith(env, `<!-- wp:comments-title {"showPostTitle":false,"textAlign":"right","level":3} /-->`)
	if !strings.Contains(out, `<h3 class="wp-block-comments-title has-text-align-right" id="comments">One response</h3>`) {
		t.Fatalf("unexpected title: %q", out)
	}
	if strings.Contains(out, "World") {
		t.Fatalf("showPostTitle=false should omit post title: %q", out)
	}
}

func TestCommentTemplatePagesRoots(t *testing.T) {
	env := &blocks.Env{
		PageComments:      true,
		CommentsPerPage:   2,
		CommentPage:       1,
		CommentTotalPages: 2,
		CommentPermalink:  "http://example.test/hello/",
		ListComments: func(postID int64) []blocks.CommentItem {
			return []blocks.CommentItem{
				{ID: 1, Author: "A", Content: "one"},
				{ID: 2, Author: "B", Content: "two"},
				{ID: 3, Author: "C", Content: "three"},
			}
		},
	}
	env.SetCurrentPost(&blocks.QueryPost{ID: 1, Title: "T"})
	out := blocks.RenderWith(env, `<!-- wp:comment-template /-->`)
	if !strings.Contains(out, `id="comment-1"`) || !strings.Contains(out, `id="comment-2"`) {
		t.Fatalf("page1=%q", out)
	}
	if strings.Contains(out, `id="comment-3"`) {
		t.Fatalf("comment-3 should be page 2: %q", out)
	}
	nav := blocks.RenderWith(env, `<!-- wp:comments-pagination -->
<!-- wp:comments-pagination-previous /-->
<!-- wp:comments-pagination-next /-->
<!-- /wp:comments-pagination -->`)
	if !strings.Contains(nav, "wp-block-comments-pagination") || !strings.Contains(nav, "cpage=2") {
		t.Fatalf("nav=%q", nav)
	}
}

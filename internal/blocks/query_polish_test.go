package blocks_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/blocks"
)

func TestQueryTagNameAndAlign(t *testing.T) {
	env := &blocks.Env{
		ListPosts: func(q blocks.QueryLoopArgs) []blocks.QueryPost {
			return []blocks.QueryPost{{ID: 1, Title: "One", Link: "/?p=1"}}
		},
	}
	in := `<!-- wp:query {"tagName":"main","align":"wide","query":{"perPage":1,"inherit":false}} -->
<!-- wp:post-template --><!-- wp:post-title /--><!-- /wp:post-template -->
<!-- /wp:query -->`
	out := blocks.RenderWith(env, in)
	if !strings.Contains(out, `<main class="wp-block-query alignwide`) {
		t.Fatalf("expected main+alignwide wrapper: %q", out)
	}
	if !strings.Contains(out, `</main>`) {
		t.Fatalf("expected closing main: %q", out)
	}
}

func TestQueryArgsOrderSearchSticky(t *testing.T) {
	var got blocks.QueryLoopArgs
	env := &blocks.Env{
		ListPosts: func(q blocks.QueryLoopArgs) []blocks.QueryPost {
			got = q
			return []blocks.QueryPost{{ID: 1, Title: "One", Link: "/?p=1"}}
		},
	}
	in := `<!-- wp:query {"query":{"perPage":2,"inherit":false,"order":"asc","orderBy":"title","search":"hello","author":"1","sticky":"ignore","exclude":[9],"format":["aside","standard"]}} -->
<div class="wp-block-query">
<!-- wp:post-template --><!-- wp:post-title /--><!-- /wp:post-template -->
</div>
<!-- /wp:query -->`
	_ = blocks.RenderWith(env, in)
	if got.Order != "asc" || got.OrderBy != "title" {
		t.Fatalf("order=%q orderBy=%q", got.Order, got.OrderBy)
	}
	if got.Search != "hello" || got.Author != 1 || got.Sticky != "ignore" {
		t.Fatalf("search/author/sticky: %+v", got)
	}
	if len(got.Exclude) != 1 || got.Exclude[0] != 9 {
		t.Fatalf("exclude=%v", got.Exclude)
	}
	if len(got.Format) != 2 {
		t.Fatalf("format=%v", got.Format)
	}
}

func TestPostTemplateLayoutClasses(t *testing.T) {
	env := &blocks.Env{
		ListPosts: func(q blocks.QueryLoopArgs) []blocks.QueryPost {
			return []blocks.QueryPost{{ID: 1, Title: "One", Link: "/?p=1"}}
		},
	}
	in := `<!-- wp:query {"query":{"perPage":1}} -->
<div class="wp-block-query">
<!-- wp:post-template {"layout":{"type":"grid","columnCount":3,"minimumColumnWidth":"12rem"},"style":{"elements":{"link":{"color":{"text":"var:preset|color|contrast"}}}}} -->
<!-- wp:post-title /-->
<!-- /wp:post-template -->
</div>
<!-- /wp:query -->`
	out := blocks.RenderWith(env, in)
	for _, want := range []string{
		`wp-block-post-template`,
		`columns-3`,
		`has-native-responsive-grid`,
		`has-link-color`,
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in %q", want, out)
		}
	}
}

func TestPostTemplateFlexDisplayLayout(t *testing.T) {
	env := &blocks.Env{
		ListPosts: func(q blocks.QueryLoopArgs) []blocks.QueryPost {
			return []blocks.QueryPost{{ID: 1, Title: "One", Link: "/?p=1"}}
		},
	}
	in := `<!-- wp:query {"query":{"perPage":1}} -->
<div class="wp-block-query">
<!-- wp:post-template {"displayLayout":{"type":"flex","columns":2}} -->
<!-- wp:post-title /-->
<!-- /wp:post-template -->
</div>
<!-- /wp:query -->`
	out := blocks.RenderWith(env, in)
	if !strings.Contains(out, `is-flex-container`) || !strings.Contains(out, `columns-2`) {
		t.Fatalf("missing flex layout classes: %q", out)
	}
}

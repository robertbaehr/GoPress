package blocks_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/blocks"
)

func TestPostTemplateClassName(t *testing.T) {
	env := &blocks.Env{
		ListPosts: func(q blocks.QueryLoopArgs) []blocks.QueryPost {
			return []blocks.QueryPost{{ID: 1, Title: "One", Link: "/?p=1"}}
		},
	}
	in := `<!-- wp:query {"query":{"perPage":1}} -->
<div class="wp-block-query">
<!-- wp:post-template {"className":"is-style-cards"} -->
<!-- wp:post-title /-->
<!-- /wp:post-template -->
</div>
<!-- /wp:query -->`
	out := blocks.RenderWith(env, in)
	if !strings.Contains(out, `wp-block-post-template`) || !strings.Contains(out, `is-style-cards`) {
		t.Fatalf("expected className on post-template: %q", out)
	}
}

func TestQueryPaginationHasLinkColorAndClassName(t *testing.T) {
	env := &blocks.Env{
		HomeURL:  func() string { return "http://example.test" },
		Paged:    1,
		MaxPages: 3,
	}
	out := blocks.RenderWith(env, `<!-- wp:query-pagination {"className":"my-pager","style":{"elements":{"link":{"color":{"text":"var:preset|color|contrast"}}}}} -->
<!-- wp:query-pagination-next /-->
<!-- /wp:query-pagination -->`)
	if !strings.Contains(out, `wp-block-query-pagination`) || !strings.Contains(out, `has-link-color`) {
		t.Fatalf("expected has-link-color: %q", out)
	}
	if !strings.Contains(out, `my-pager`) {
		t.Fatalf("expected className: %q", out)
	}
}

func TestQueryPaginationArrowNoneAndChevron(t *testing.T) {
	env := &blocks.Env{
		HomeURL:  func() string { return "http://example.test" },
		Paged:    2,
		MaxPages: 5,
	}
	none := blocks.RenderWith(env, `<!-- wp:query-pagination {"paginationArrow":"none"} -->
<!-- wp:query-pagination-previous /-->
<!-- wp:query-pagination-next /-->
<!-- /wp:query-pagination -->`)
	if strings.Contains(none, "is-arrow-") {
		t.Fatalf("paginationArrow=none should omit arrows: %q", none)
	}
	if !strings.Contains(none, "Previous Page") || !strings.Contains(none, "Next Page") {
		t.Fatalf("expected labels: %q", none)
	}

	chev := blocks.RenderWith(env, `<!-- wp:query-pagination {"paginationArrow":"chevron"} -->
<!-- wp:query-pagination-previous /-->
<!-- wp:query-pagination-next /-->
<!-- /wp:query-pagination -->`)
	if !strings.Contains(chev, `is-arrow-chevron`) || !strings.Contains(chev, "«") || !strings.Contains(chev, "»") {
		t.Fatalf("expected chevron arrows: %q", chev)
	}
}

func TestQueryPaginationShowLabelFalse(t *testing.T) {
	env := &blocks.Env{
		HomeURL:  func() string { return "http://example.test" },
		Paged:    2,
		MaxPages: 5,
	}
	out := blocks.RenderWith(env, `<!-- wp:query-pagination {"showLabel":false,"paginationArrow":"arrow"} -->
<!-- wp:query-pagination-previous /-->
<!-- wp:query-pagination-next /-->
<!-- /wp:query-pagination -->`)
	if !strings.Contains(out, `aria-label="Previous Page"`) || !strings.Contains(out, `aria-label="Next Page"`) {
		t.Fatalf("expected aria-label when label hidden: %q", out)
	}
	// Label text must not appear as link body (only aria-label).
	if strings.Contains(out, `>Previous Page<`) || strings.Contains(out, `>Next Page<`) {
		t.Fatalf("showLabel=false should hide label text body: %q", out)
	}
	if !strings.Contains(out, `is-arrow-arrow`) {
		t.Fatalf("expected arrow spans: %q", out)
	}
}

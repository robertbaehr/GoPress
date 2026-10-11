package blocks_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/blocks"
)

func TestQueryPaginationAnchorFlexAndColors(t *testing.T) {
	env := &blocks.Env{
		HomeURL:  func() string { return "http://example.test" },
		Paged:    1,
		MaxPages: 3,
	}
	out := blocks.RenderWith(env, `<!-- wp:query-pagination {"anchor":"pager-top","backgroundColor":"base","textColor":"contrast","className":"my-pager"} -->
<!-- wp:query-pagination-next /-->
<!-- /wp:query-pagination -->`)
	for _, want := range []string{
		`id="pager-top"`,
		`is-layout-flex`,
		`wp-block-query-pagination-is-layout-flex`,
		`has-background`,
		`has-base-background-color`,
		`has-text-color`,
		`has-contrast-color`,
		`my-pager`,
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in %q", want, out)
		}
	}
}

func TestQueryPaginationNumbersMidSizeAndClassName(t *testing.T) {
	env := &blocks.Env{
		HomeURL:  func() string { return "http://example.test" },
		Paged:    3,
		MaxPages: 10,
	}
	out := blocks.RenderWith(env, `<!-- wp:query-pagination -->
<!-- wp:query-pagination-numbers {"midSize":1,"className":"nums","anchor":"nums-id"} /-->
<!-- /wp:query-pagination -->`)
	if !strings.Contains(out, `id="nums-id"`) || !strings.Contains(out, `nums`) {
		t.Fatalf("expected numbers wrapper attrs: %q", out)
	}
	// midSize=1 around page 3 → window 2,3,4 (plus 1 and 10).
	if !strings.Contains(out, `>2<`) || !strings.Contains(out, `>4<`) {
		t.Fatalf("expected midSize window: %q", out)
	}
	// Page 5 should be outside midSize=1 window (dots instead).
	if strings.Contains(out, `>5</a>`) {
		t.Fatalf("midSize=1 should omit page 5 link: %q", out)
	}
}

func TestQueryPaginationPrevNextClassName(t *testing.T) {
	env := &blocks.Env{
		HomeURL:  func() string { return "http://example.test" },
		Paged:    2,
		MaxPages: 4,
	}
	out := blocks.RenderWith(env, `<!-- wp:query-pagination {"paginationArrow":"none"} -->
<!-- wp:query-pagination-previous {"className":"prev-extra"} /-->
<!-- wp:query-pagination-next {"className":"next-extra"} /-->
<!-- /wp:query-pagination -->`)
	if !strings.Contains(out, `wp-block-query-pagination-previous prev-extra`) {
		t.Fatalf("expected previous className: %q", out)
	}
	if !strings.Contains(out, `wp-block-query-pagination-next next-extra`) {
		t.Fatalf("expected next className: %q", out)
	}
}

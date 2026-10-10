package blocks_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/blocks"
)

func TestQueryPaginationPage1(t *testing.T) {
	env := &blocks.Env{
		HomeURL:  func() string { return "http://example.test" },
		Paged:    1,
		MaxPages: 39,
	}
	out := blocks.RenderWith(env, `<!-- wp:query-pagination -->
<!-- wp:query-pagination-previous /-->
<!-- wp:query-pagination-numbers /-->
<!-- wp:query-pagination-next /-->
<!-- /wp:query-pagination -->`)
	if strings.Contains(out, "query-pagination-previous") {
		t.Fatalf("page 1 should omit previous: %q", out)
	}
	if !strings.Contains(out, `class="page-numbers current"`) || !strings.Contains(out, ">1</span>") {
		t.Fatalf("expected current page 1: %q", out)
	}
	if !strings.Contains(out, `href="http://example.test/page/2/"`) {
		t.Fatalf("expected next/page 2 link: %q", out)
	}
	if !strings.Contains(out, "query-pagination-next") || !strings.Contains(out, "Next Page") {
		t.Fatalf("expected next: %q", out)
	}
	if !strings.Contains(out, `class="page-numbers dots"`) || !strings.Contains(out, "/page/39/") {
		t.Fatalf("expected dots + last page: %q", out)
	}
}

func TestQueryPaginationPage2(t *testing.T) {
	env := &blocks.Env{
		HomeURL:  func() string { return "http://example.test" },
		Paged:    2,
		MaxPages: 39,
	}
	out := blocks.RenderWith(env, `<!-- wp:query-pagination-previous /-->`)
	if !strings.Contains(out, "query-pagination-previous") || !strings.Contains(out, "Previous Page") {
		t.Fatalf("expected previous: %q", out)
	}
	if !strings.Contains(out, `href="http://example.test/"`) {
		t.Fatalf("previous should link home: %q", out)
	}
	nums := blocks.RenderWith(env, `<!-- wp:query-pagination-numbers /-->`)
	if !strings.Contains(nums, `aria-current="page" class="page-numbers current">2</span>`) {
		t.Fatalf("expected current 2: %q", nums)
	}
	next := blocks.RenderWith(env, `<!-- wp:query-pagination-next /-->`)
	if !strings.Contains(next, "/page/3/") {
		t.Fatalf("expected next page 3: %q", next)
	}
}

func TestQueryPaginationLastPageOmitsNext(t *testing.T) {
	env := &blocks.Env{
		HomeURL:  func() string { return "http://example.test" },
		Paged:    5,
		MaxPages: 5,
	}
	next := blocks.RenderWith(env, `<!-- wp:query-pagination-next /-->`)
	if strings.TrimSpace(next) != "" {
		t.Fatalf("last page should omit next: %q", next)
	}
}

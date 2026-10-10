package wp_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestPaginateLinksBasic(t *testing.T) {
	out := wp.PaginateLinks(wp.PaginateLinksArgs{
		Base:    "http://example.test/%_%",
		Format:  "page/%#%/",
		Total:   5,
		Current: 1,
		EndSize: 1,
		MidSize: 2,
	})
	if out == "" {
		t.Fatal("expected markup")
	}
	if !strings.Contains(out, `aria-current="page"`) || !strings.Contains(out, `class="page-numbers current"`) {
		t.Fatalf("current page: %q", out)
	}
	if strings.Contains(out, `class="prev page-numbers"`) {
		t.Fatalf("page 1 should omit prev: %q", out)
	}
	if !strings.Contains(out, `class="next page-numbers"`) || !strings.Contains(out, `href="http://example.test/page/2/"`) {
		t.Fatalf("expected next→page/2: %q", out)
	}
}

func TestPaginateLinksDotsAndPrev(t *testing.T) {
	out := wp.PaginateLinks(wp.PaginateLinksArgs{
		Base:    "/blog/%_%",
		Format:  "page/%#%/",
		Total:   10,
		Current: 5,
		EndSize: 1,
		MidSize: 1,
	})
	if !strings.Contains(out, `class="page-numbers dots"`) {
		t.Fatalf("expected dots: %q", out)
	}
	if !strings.Contains(out, `class="prev page-numbers"`) || !strings.Contains(out, `href="/blog/page/4/"`) {
		t.Fatalf("expected prev: %q", out)
	}
	if !strings.Contains(out, `href="/blog/"`) && !strings.Contains(out, `href="/blog"`) {
		// page 1 strips format → base with empty %_%
		if !strings.Contains(out, `>1</a>`) {
			t.Fatalf("expected page 1 link: %q", out)
		}
	}
}

func TestPaginateLinksNoPrevNextAndList(t *testing.T) {
	if wp.PaginateLinks(wp.PaginateLinksArgs{Total: 1, Current: 1}) != "" {
		t.Fatal("total<2 should be empty")
	}
	out := wp.PaginateLinks(wp.PaginateLinksArgs{
		Base:       "/?%",
		Format:     "paged=%#%",
		Total:      3,
		Current:    2,
		NoPrevNext: true,
		Type:       "list",
		ShowAll:    true,
	})
	if strings.Contains(out, "prev ") || strings.Contains(out, "next ") {
		t.Fatalf("NoPrevNext: %q", out)
	}
	if !strings.HasPrefix(out, "<ul class='page-numbers'>") {
		t.Fatalf("list type: %q", out)
	}
}

func TestPaginateLinksPlain(t *testing.T) {
	out := wp.PaginateLinksPlain("http://example.test/tag/news", 2, 4)
	if !strings.Contains(out, `href="http://example.test/tag/news/"`) {
		t.Fatalf("page1 base: %q", out)
	}
	if !strings.Contains(out, `href="http://example.test/tag/news/page/3/"`) {
		t.Fatalf("next page3: %q", out)
	}
}

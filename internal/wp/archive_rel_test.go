package wp_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestGetPagenumLink(t *testing.T) {
	base := "http://example.com/category/news/"
	if got := wp.GetPagenumLink(base, 1); got != base {
		t.Fatalf("page1=%q", got)
	}
	if got := wp.GetPagenumLink(base, 2); got != "http://example.com/category/news/page/2/" {
		t.Fatalf("page2=%q", got)
	}
	if got := wp.GetPagenumLink("http://example.com/tag/x", 3); got != "http://example.com/tag/x/page/3/" {
		t.Fatalf("page3=%q", got)
	}
}

func TestArchiveRelPrevNextHTML(t *testing.T) {
	base := "http://example.com/category/news/"
	if got := wp.ArchiveRelPrevNextHTML(base, 1, 1); got != "" {
		t.Fatalf("single page should be empty: %q", got)
	}
	page1 := wp.ArchiveRelPrevNextHTML(base, 1, 3)
	if strings.Contains(page1, `rel="prev"`) || !strings.Contains(page1, `rel="next"`) || !strings.Contains(page1, "/page/2/") {
		t.Fatalf("page1: %q", page1)
	}
	page2 := wp.ArchiveRelPrevNextHTML(base, 2, 3)
	if !strings.Contains(page2, `rel="prev"`) || !strings.Contains(page2, base) ||
		!strings.Contains(page2, `rel="next"`) || !strings.Contains(page2, "/page/3/") {
		t.Fatalf("page2: %q", page2)
	}
	page3 := wp.ArchiveRelPrevNextHTML(base, 3, 3)
	if !strings.Contains(page3, `rel="prev"`) || !strings.Contains(page3, "/page/2/") ||
		strings.Contains(page3, `rel="next"`) {
		t.Fatalf("page3: %q", page3)
	}
}

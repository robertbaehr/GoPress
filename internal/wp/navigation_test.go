package wp_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestLoadNavigationContent(t *testing.T) {
	c := bootWP(t)
	content, ok := wp.LoadNavigationContent(c, 446)
	if !ok {
		t.Fatal("expected fixture-navigation")
	}
	if !strings.Contains(content, "navigation-link") || !strings.Contains(content, "Home") {
		t.Fatalf("got %q", content)
	}
}

func TestListPublishedPages(t *testing.T) {
	c := bootWP(t)
	pages, err := wp.ListPublishedPages(c, 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(pages) == 0 {
		t.Fatal("expected pages")
	}
	found := false
	for _, p := range pages {
		if p.Name == "sample-page" || strings.Contains(p.Title, "Sample Page") {
			found = true
			link := wp.PagePermalink(c, p)
			if !strings.Contains(link, "sample-page") && !strings.Contains(link, "page_id=") {
				t.Fatalf("permalink=%q", link)
			}
			break
		}
	}
	if !found {
		t.Fatalf("sample-page missing: %+v", pages)
	}
}

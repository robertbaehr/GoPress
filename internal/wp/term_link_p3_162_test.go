package wp_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// P3.162: GetTermLink / GetCategoryLink / GetTagLink pretty paths + filters.
func TestGetTermLinkP3162(t *testing.T) {
	c := testBootQuery(t)
	origPerm := c.Options.GetOption("permalink_structure", "")
	origHome := c.Options.GetOption("home", "")
	t.Cleanup(func() {
		_, _ = c.Options.UpdateOption("permalink_structure", origPerm, "yes")
		_, _ = c.Options.UpdateOption("home", origHome, "yes")
	})
	_, _ = c.Options.UpdateOption("permalink_structure", "/%year%/%monthnum%/%day%/%postname%/", "yes")
	_, _ = c.Options.UpdateOption("home", "http://example.test", "yes")

	cat := wp.GetCategoryLink(c, 1)
	if cat != "http://example.test/category/uncategorized/" {
		t.Fatalf("category=%q", cat)
	}
	tag := wp.GetTagLink(c, 5)
	if tag != "http://example.test/tag/fixture-tag-01/" {
		t.Fatalf("tag=%q", tag)
	}
	if got := wp.GetTermLink(c, 1, "category"); got != cat {
		t.Fatalf("alias=%q", got)
	}
	if got := wp.GetCategoryLink(c, 0); got != "" {
		t.Fatalf("zero=%q", got)
	}

	c.Hooks.AddFilter("category_link", func(value any, _ ...any) any {
		return "http://example.test/custom-cat/"
	}, 10, 2)
	c.Hooks.AddFilter("term_link", func(value any, _ ...any) any {
		return "http://example.test/term-filtered/"
	}, 10, 3)
	if got := wp.GetCategoryLink(c, 1); got != "http://example.test/term-filtered/" {
		t.Fatalf("filtered=%q", got)
	}
}

// P3.162: query-arg form when permalink_structure empty.
func TestGetTermLinkQueryP3162(t *testing.T) {
	c := testBootQuery(t)
	origPerm := c.Options.GetOption("permalink_structure", "")
	origHome := c.Options.GetOption("home", "")
	t.Cleanup(func() {
		_, _ = c.Options.UpdateOption("permalink_structure", origPerm, "yes")
		_, _ = c.Options.UpdateOption("home", origHome, "yes")
	})
	_, _ = c.Options.UpdateOption("permalink_structure", "", "yes")
	_, _ = c.Options.UpdateOption("home", "http://example.test", "yes")

	if got := wp.GetCategoryLink(c, 1); got != "http://example.test/?cat=1" {
		t.Fatalf("cat query=%q", got)
	}
	if got := wp.GetTagLink(c, 5); !strings.Contains(got, "tag=fixture-tag-01") {
		t.Fatalf("tag query=%q", got)
	}
}

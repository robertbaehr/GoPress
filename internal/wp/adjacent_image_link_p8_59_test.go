package wp_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

func parentAttachment(t *testing.T, c *wp.Context, id, parent int64) {
	t.Helper()
	q, args, err := c.DB.Prepare("UPDATE "+c.DB.Posts()+" SET post_parent = %d WHERE ID = %d", parent, id)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.DB.Exec(q, args...); err != nil {
		t.Fatal(err)
	}
	if c.Cache != nil {
		c.Cache.Flush()
	}
}

// P8.59: GetAdjacentImageLink walks sibling images under the same parent.
func TestGetAdjacentImageLinkP859(t *testing.T) {
	c := testBootQuery(t)
	parentAttachment(t, c, 5, 1)
	parentAttachment(t, c, 6, 1)
	t.Cleanup(func() {
		parentAttachment(t, c, 5, 0)
		parentAttachment(t, c, 6, 0)
	})

	prev := wp.GetPreviousImageLink(c, 6, "thumbnail", "")
	if !strings.Contains(prev, "attachment_id=5") && !strings.Contains(prev, `href=`) {
		t.Fatalf("previous=%q", prev)
	}
	next := wp.GetNextImageLink(c, 5, "thumbnail", "")
	if !strings.Contains(next, "attachment_id=6") && !strings.Contains(next, `href=`) {
		t.Fatalf("next=%q", next)
	}
	// No previous for first sibling.
	if got := wp.GetPreviousImageLink(c, 5, "thumbnail", ""); got != "" {
		t.Fatalf("first previous should be empty: %q", got)
	}

	c.Hooks.AddFilter("next_image_link", func(value any, _ ...any) any {
		return "FILTERED-NEXT"
	}, 10, 4)
	if got := wp.GetNextImageLink(c, 5, "thumbnail", ""); got != "FILTERED-NEXT" {
		t.Fatalf("filter=%q", got)
	}
}

// P8.59: text override builds a text link.
func TestGetAdjacentImageLinkTextP859(t *testing.T) {
	c := testBootQuery(t)
	parentAttachment(t, c, 5, 1)
	parentAttachment(t, c, 6, 1)
	t.Cleanup(func() {
		parentAttachment(t, c, 5, 0)
		parentAttachment(t, c, 6, 0)
	})
	got := wp.GetNextImageLink(c, 5, "thumbnail", "Next →")
	if !strings.Contains(got, "Next →") || !strings.Contains(got, "<a href=") {
		t.Fatalf("text link=%q", got)
	}
}

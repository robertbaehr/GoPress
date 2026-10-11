package wp_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestGetAdjacentPostPreviousNext(t *testing.T) {
	c := testBootQuery(t)

	// Fixture posts are dated; pick a mid post and walk neighbors.
	mid, err := wp.GetPostByID(c, 13)
	if err != nil || mid == nil || mid.Status != "publish" {
		t.Skip("fixture post 13 missing")
	}
	prev, err := wp.GetPreviousPost(c, mid.ID, "")
	if err != nil {
		t.Fatalf("previous: %v", err)
	}
	next, err := wp.GetNextPost(c, mid.ID, "")
	if err != nil {
		t.Fatalf("next: %v", err)
	}
	if prev == nil && next == nil {
		t.Fatal("expected at least one adjacent published post")
	}
	if prev != nil && prev.Date > mid.Date {
		t.Fatalf("previous date %q after mid %q", prev.Date, mid.Date)
	}
	if next != nil && next.Date < mid.Date {
		t.Fatalf("next date %q before mid %q", next.Date, mid.Date)
	}
}

func TestGetAdjacentPostWhereFilter(t *testing.T) {
	c := testBootQuery(t)
	if c.Hooks == nil {
		c.Hooks = wp.NewHooks()
	}
	// Force empty result via WHERE filter (keep placeholders for Prepare).
	c.Hooks.AddFilter("get_previous_post_where", func(value any, args ...any) any {
		s, _ := value.(string)
		return s + " AND 1=0"
	}, 10, 1)
	got, err := wp.GetPreviousPost(c, 13, "")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if got != nil {
		t.Fatalf("expected nil after WHERE 1=0, got %#v", got)
	}
}

func TestAdjacentPostPermalink(t *testing.T) {
	adj := &wp.AdjacentPostResult{ID: 7, Name: "hello-world", Date: "2024-06-15 12:00:00"}
	link := wp.AdjacentPostPermalink("http://example.com", adj)
	if !strings.Contains(link, "/2024/06/15/hello-world/") {
		t.Fatalf("pretty permalink: %q", link)
	}
	bare := wp.AdjacentPostPermalink("http://example.com", &wp.AdjacentPostResult{ID: 9})
	if bare != "http://example.com/?p=9" {
		t.Fatalf("fallback: %q", bare)
	}
}

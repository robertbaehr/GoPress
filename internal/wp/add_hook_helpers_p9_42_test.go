package wp_test

import (
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// P9.42: AddAction / AddFilter Context helpers register on c.Hooks.
func TestAddActionAddFilterHelpersP942(t *testing.T) {
	c := testBootQuery(t)
	var acted int
	wp.AddAction(c, "p942_action", func(_ ...any) { acted++ }, 10, 0)
	c.Hooks.DoAction("p942_action")
	if acted != 1 {
		t.Fatalf("acted=%d", acted)
	}
	if !wp.HasAction(c, "p942_action") {
		t.Fatal("HasAction")
	}

	wp.AddFilter(c, "p942_filter", func(value any, _ ...any) any {
		if s, ok := value.(string); ok {
			return s + "-ok"
		}
		return value
	}, 10, 1)
	got := c.Hooks.ApplyFilters("p942_filter", "seed")
	if got != "seed-ok" {
		t.Fatalf("got=%v", got)
	}
	if !wp.HasFilter(c, "p942_filter") {
		t.Fatal("HasFilter")
	}

	wp.AddAction(nil, "x", func(_ ...any) {}, 10, 0)
	wp.AddFilter(nil, "x", func(value any, _ ...any) any { return value }, 10, 1)
}

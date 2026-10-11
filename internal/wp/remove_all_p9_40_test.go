package wp_test

import (
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// P9.40: RemoveAllFilters / RemoveAllActions Context helpers.
func TestRemoveAllFiltersActionsP940(t *testing.T) {
	if wp.RemoveAllFilters(nil, "t", -1) || wp.RemoveAllActions(nil, "a", -1) {
		t.Fatal("nil context")
	}
	c := &wp.Context{Hooks: wp.NewHooks()}
	if wp.RemoveAllFilters(c, "", -1) || wp.RemoveAllActions(c, "", -1) {
		t.Fatal("empty tag")
	}
	ffn := func(value any, _ ...any) any { return value }
	afn := func(_ ...any) {}
	c.Hooks.AddFilter("t", ffn, 10, 1)
	c.Hooks.AddFilter("t", ffn, 20, 1)
	c.Hooks.AddAction("a", afn, 10, 0)
	if !wp.RemoveAllFilters(c, "t", 10) {
		t.Fatal("RemoveAllFilters priority")
	}
	if !wp.HasFilter(c, "t") {
		t.Fatal("priority 20 should remain")
	}
	if !wp.RemoveAllFilters(c, "t", -1) {
		t.Fatal("RemoveAllFilters all")
	}
	if wp.HasFilter(c, "t") {
		t.Fatal("filters should be empty")
	}
	if !wp.RemoveAllActions(c, "a", -1) {
		t.Fatal("RemoveAllActions")
	}
	if wp.HasAction(c, "a") {
		t.Fatal("actions should be empty")
	}
}

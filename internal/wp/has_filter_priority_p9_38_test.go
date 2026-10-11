package wp_test

import (
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// P9.38: HasFilterPriority package helper mirrors has_filter with callback.
func TestHasFilterPriorityP938(t *testing.T) {
	if _, ok := wp.HasFilterPriority(nil, "t", nil, -1); ok {
		t.Fatal("nil context")
	}
	c := &wp.Context{Hooks: wp.NewHooks()}
	fn := func(value any, _ ...any) any { return value }
	if _, ok := wp.HasFilterPriority(c, "", fn, -1); ok {
		t.Fatal("empty tag")
	}
	if _, ok := wp.HasFilterPriority(c, "t", nil, -1); ok {
		t.Fatal("nil fn")
	}
	if _, ok := wp.HasFilterPriority(c, "t", fn, -1); ok {
		t.Fatal("unregistered")
	}
	c.Hooks.AddFilter("t", fn, 20, 1)
	pri, ok := wp.HasFilterPriority(c, "t", fn, -1)
	if !ok || pri != 20 {
		t.Fatalf("any: pri=%d ok=%v", pri, ok)
	}
	if _, ok := wp.HasFilterPriority(c, "t", fn, 10); ok {
		t.Fatal("wrong priorityExact")
	}
	if pri, ok := wp.HasFilterPriority(c, "t", fn, 20); !ok || pri != 20 {
		t.Fatalf("exact: pri=%d ok=%v", pri, ok)
	}
}

// P9.38: RemoveFilter / RemoveAction package helpers.
func TestRemoveFilterActionP938(t *testing.T) {
	if wp.RemoveFilter(nil, "t", nil, 10) || wp.RemoveAction(nil, "a", nil, 10) {
		t.Fatal("nil context")
	}
	c := &wp.Context{Hooks: wp.NewHooks()}
	ffn := func(value any, _ ...any) any { return value }
	afn := func(_ ...any) {}
	if wp.RemoveFilter(c, "t", ffn, 10) || wp.RemoveAction(c, "a", afn, 10) {
		t.Fatal("unregistered remove")
	}
	c.Hooks.AddFilter("t", ffn, 10, 1)
	c.Hooks.AddAction("a", afn, 10, 0)
	if !wp.RemoveFilter(c, "t", ffn, 10) {
		t.Fatal("RemoveFilter")
	}
	if wp.RemoveFilter(c, "t", ffn, 10) {
		t.Fatal("second RemoveFilter")
	}
	if !wp.RemoveAction(c, "a", afn, 10) {
		t.Fatal("RemoveAction")
	}
	if wp.RemoveAction(c, "a", afn, 10) {
		t.Fatal("second RemoveAction")
	}
	if wp.HasFilter(c, "t") || wp.HasAction(c, "a") {
		t.Fatal("hooks should be empty")
	}
}

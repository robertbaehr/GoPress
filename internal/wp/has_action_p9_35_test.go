package wp_test

import (
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// P9.35: package-level has_action / has_filter; empty tag; shared registry.
func TestHasActionPackageHelpersP935(t *testing.T) {
	if wp.HasAction(nil, "init") {
		t.Fatal("nil context")
	}
	c := &wp.Context{Hooks: wp.NewHooks()}
	if wp.HasAction(c, "") || wp.HasFilter(c, "") {
		t.Fatal("empty tag")
	}
	if wp.HasAction(c, "init") || c.Hooks.HasAction("") {
		t.Fatal("miss before add")
	}

	fn := func(args ...any) {}
	c.Hooks.AddAction("init", fn, 10, 0)
	if !wp.HasAction(c, "init") || !wp.HasFilter(c, "init") {
		t.Fatal("shared registry: action visible to HasFilter")
	}
	pri, ok := wp.HasActionPriority(c, "init", fn, -1)
	if !ok || pri != 10 {
		t.Fatalf("HasActionPriority: pri=%d ok=%v", pri, ok)
	}
	if _, ok := wp.HasActionPriority(c, "init", fn, 20); ok {
		t.Fatal("exact priority miss")
	}

	// Filter registration also makes HasAction true.
	c.Hooks.AddFilter("the_title", func(value any, _ ...any) any { return value }, 5, 1)
	if !wp.HasAction(c, "the_title") {
		t.Fatal("filter should make HasAction true")
	}
}

func TestHasActionAfterRemoveAllP935(t *testing.T) {
	h := wp.NewHooks()
	fn := func(args ...any) {}
	h.AddAction("shutdown", fn, 10, 0)
	h.AddAction("shutdown", fn, 20, 0)
	if !h.HasAction("shutdown") {
		t.Fatal("expected true")
	}
	pri, ok := h.HasActionFunc("shutdown", fn, -1)
	if !ok || pri != 10 {
		t.Fatalf("lowest priority first: pri=%d ok=%v", pri, ok)
	}
	h.RemoveAllActions("shutdown", 10)
	if !h.HasAction("shutdown") {
		t.Fatal("priority 20 should remain")
	}
	pri, ok = h.HasActionFunc("shutdown", fn, -1)
	if !ok || pri != 20 {
		t.Fatalf("after remove pri10: pri=%d ok=%v", pri, ok)
	}
	h.RemoveAllActions("shutdown", -1)
	if h.HasAction("shutdown") {
		t.Fatal("expected cleared")
	}
}

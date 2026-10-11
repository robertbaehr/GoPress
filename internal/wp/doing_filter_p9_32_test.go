package wp_test

import (
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// P9.32: 'all' runs first while doing_filter(tag) is true; 'all' itself is not on the stack.
func TestDoingFilterAllHookP932(t *testing.T) {
	h := wp.NewHooks()
	var allSawDoing bool
	var allSawCurrent string
	h.AddAction("all", func(args ...any) {
		if len(args) < 1 {
			t.Fatal("all expects tag arg")
		}
		tag, _ := args[0].(string)
		allSawCurrent = h.CurrentFilter()
		allSawDoing = h.DoingFilter(tag) && h.DoingFilter("")
		if h.DoingFilter("all") {
			t.Error("doing_filter('all') should be false; tag is on the stack, not 'all'")
		}
	}, 10, 3)
	h.AddFilter("the_title", func(value any, args ...any) any {
		if !h.DoingFilter("the_title") {
			t.Error("expected doing the_title during filter")
		}
		return value.(string) + "-ok"
	}, 10, 1)

	got := h.ApplyFilters("the_title", "Hi")
	if got != "Hi-ok" {
		t.Fatalf("got %v", got)
	}
	if !allSawDoing || allSawCurrent != "the_title" {
		t.Fatalf("all hook doing=%v current=%q", allSawDoing, allSawCurrent)
	}
	if h.DoingFilter("") || h.DoingFilter("the_title") {
		t.Fatal("stack should be clear")
	}
}

// P9.32: DoAction with only 'all' still sets doing_filter during the all callback.
func TestDoingFilterAllOnlyActionP932(t *testing.T) {
	h := wp.NewHooks()
	var saw bool
	h.AddAction("all", func(args ...any) {
		saw = h.DoingAction("my_action") && h.DoingFilter("my_action")
	}, 10, 2)
	h.DoAction("my_action", 1)
	if !saw {
		t.Fatal("expected doing_filter during all-only action")
	}
	if h.DidAction("my_action") != 1 {
		t.Fatalf("did_action=%d", h.DidAction("my_action"))
	}
}

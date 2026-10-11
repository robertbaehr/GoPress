package wp_test

import (
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// P9.37: package DoingAction / DidAction deepen nested stack + counters.
func TestDoingActionNestedDidActionP937(t *testing.T) {
	if wp.DoingAction(nil, "x") || wp.DidAction(nil, "x") != 0 {
		t.Fatal("nil context")
	}
	if wp.DidFilter(nil, "f") != 0 {
		t.Fatal("nil DidFilter")
	}

	c := &wp.Context{Hooks: wp.NewHooks()}
	if wp.DoingAction(c, "") || wp.DidAction(c, "outer") != 0 {
		t.Fatal("empty before fire")
	}

	var midDoingOuter, midDoingInner, midDoingEmpty bool
	var midDidOuter, midDidInner int
	c.Hooks.AddAction("outer", func(_ ...any) {
		if midDidOuter == 0 {
			midDoingOuter = wp.DoingAction(c, "outer")
			midDoingEmpty = wp.DoingAction(c, "")
			midDidOuter = wp.DidAction(c, "outer")
		}
		c.Hooks.DoAction("inner")
		if midDidInner == 0 {
			midDidInner = wp.DidAction(c, "inner")
		}
	}, 10, 0)
	c.Hooks.AddAction("inner", func(_ ...any) {
		if !midDoingInner {
			midDoingInner = wp.DoingAction(c, "inner")
		}
		if !wp.DoingAction(c, "outer") {
			t.Error("DoingAction(outer) false while inner runs")
		}
		if n := wp.DidAction(c, "inner"); n < 1 {
			t.Errorf("DidAction(inner) mid=%d", n)
		}
	}, 10, 0)

	c.Hooks.DoAction("outer")
	c.Hooks.DoAction("outer")

	if !midDoingOuter || !midDoingInner || !midDoingEmpty {
		t.Fatalf("mid doing outer=%v inner=%v empty=%v", midDoingOuter, midDoingInner, midDoingEmpty)
	}
	// did_action bumps before callbacks run; first outer fire → 1.
	if midDidOuter != 1 {
		t.Fatalf("midDidOuter=%d", midDidOuter)
	}
	if midDidInner != 1 {
		t.Fatalf("midDidInner=%d", midDidInner)
	}
	if wp.DidAction(c, "outer") != 2 {
		t.Fatalf("DidAction(outer)=%d", wp.DidAction(c, "outer"))
	}
	if wp.DidAction(c, "inner") != 2 {
		t.Fatalf("DidAction(inner)=%d", wp.DidAction(c, "inner"))
	}
	if wp.DoingAction(c, "outer") || wp.DoingAction(c, "") {
		t.Fatal("stack should clear after DoAction")
	}
}

func TestDoingActionSharedWithFilterP937(t *testing.T) {
	c := &wp.Context{Hooks: wp.NewHooks()}
	var sawDoing int
	var firstDidFilter int
	c.Hooks.AddFilter("f", func(v any, _ ...any) any {
		if !wp.DoingAction(c, "f") || !wp.DoingFilter(c, "f") {
			t.Error("DoingAction/Filter should share stack")
		}
		sawDoing++
		if firstDidFilter == 0 {
			firstDidFilter = wp.DidFilter(c, "f")
		}
		return v
	}, 10, 1)
	_ = c.Hooks.ApplyFilters("f", "x")
	_ = c.Hooks.ApplyFilters("f", "y")
	if sawDoing != 2 {
		t.Fatalf("sawDoing=%d", sawDoing)
	}
	if wp.DidFilter(c, "f") != 2 {
		t.Fatalf("DidFilter=%d", wp.DidFilter(c, "f"))
	}
	if firstDidFilter != 1 {
		t.Fatalf("first DidFilter=%d", firstDidFilter)
	}
	if wp.DidAction(c, "f") != 0 {
		t.Fatalf("ApplyFilters should not bump DidAction: %d", wp.DidAction(c, "f"))
	}
}

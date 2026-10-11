package wp_test

import (
	"reflect"
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// P9.36: package CurrentAction/CurrentFilter/DoingAction helpers.
func TestCurrentActionPackageHelpersP936(t *testing.T) {
	if wp.CurrentAction(nil) != "" || wp.CurrentFilter(nil) != "" {
		t.Fatal("nil context")
	}
	if wp.DoingAction(nil, "") || wp.DoingFilter(nil, "x") {
		t.Fatal("nil doing")
	}
	if wp.CurrentFilters(nil) != nil {
		t.Fatal("nil filters")
	}

	c := &wp.Context{Hooks: wp.NewHooks()}
	if wp.CurrentAction(c) != "" || wp.DoingAction(c, "") {
		t.Fatal("empty stack")
	}

	var sawAction, sawFilter string
	var sawDoingOuter, sawDoingEmpty bool
	var stack []string
	c.Hooks.AddAction("outer", func(_ ...any) {
		sawAction = wp.CurrentAction(c)
		sawFilter = wp.CurrentFilter(c)
		sawDoingOuter = wp.DoingAction(c, "outer")
		sawDoingEmpty = wp.DoingAction(c, "")
		stack = wp.CurrentFilters(c)
		c.Hooks.DoAction("inner")
		if wp.CurrentAction(c) != "outer" {
			t.Errorf("after nested CurrentAction=%q", wp.CurrentAction(c))
		}
	}, 10, 0)
	c.Hooks.AddAction("inner", func(_ ...any) {
		if wp.CurrentAction(c) != "inner" || !wp.DoingAction(c, "inner") {
			t.Errorf("inner CurrentAction=%q", wp.CurrentAction(c))
		}
		if !wp.DoingFilter(c, "outer") {
			t.Error("DoingFilter(outer) while nested inner")
		}
	}, 10, 0)
	c.Hooks.DoAction("outer")

	if sawAction != "outer" || sawFilter != "outer" {
		t.Fatalf("saw action=%q filter=%q", sawAction, sawFilter)
	}
	if !sawDoingOuter || !sawDoingEmpty {
		t.Fatal("DoingAction outer/empty")
	}
	if !reflect.DeepEqual(stack, []string{"outer"}) {
		t.Fatalf("stack=%v", stack)
	}
	if wp.CurrentAction(c) != "" || wp.DoingAction(c, "outer") {
		t.Fatal("cleared after DoAction")
	}
}

func TestCurrentActionSharedRegistryP936(t *testing.T) {
	c := &wp.Context{Hooks: wp.NewHooks()}
	var got string
	c.Hooks.AddFilter("f", func(v any, _ ...any) any {
		got = wp.CurrentFilter(c)
		if wp.CurrentAction(c) != got {
			t.Fatalf("CurrentAction=%q CurrentFilter=%q", wp.CurrentAction(c), got)
		}
		return v
	}, 10, 1)
	_ = c.Hooks.ApplyFilters("f", "x")
	if got != "f" {
		t.Fatalf("got=%q", got)
	}
}

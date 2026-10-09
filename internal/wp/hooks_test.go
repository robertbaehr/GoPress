package wp_test

import (
	"fmt"
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestHooksPriorityOrder(t *testing.T) {
	h := wp.NewHooks()
	var got []int
	h.AddAction("x", func(args ...any) { got = append(got, 20) }, 20, 0)
	h.AddAction("x", func(args ...any) { got = append(got, 5) }, 5, 0)
	h.AddAction("x", func(args ...any) { got = append(got, 10) }, 10, 0)
	h.DoAction("x")
	want := []int{5, 10, 20}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("order=%v want %v", got, want)
	}
}

func TestHooksStableEqualPriority(t *testing.T) {
	h := wp.NewHooks()
	var got []string
	h.AddAction("x", func(args ...any) { got = append(got, "a") }, 10, 0)
	h.AddAction("x", func(args ...any) { got = append(got, "b") }, 10, 0)
	h.AddAction("x", func(args ...any) { got = append(got, "c") }, 10, 0)
	h.DoAction("x")
	if fmt.Sprint(got) != "[a b c]" {
		t.Fatalf("got %v", got)
	}
}

func namedActionA(args ...any) {
	if len(args) == 1 {
		if p, ok := args[0].(*[]string); ok {
			*p = append(*p, "A")
		}
	}
}

func namedActionB(args ...any) {
	if len(args) == 1 {
		if p, ok := args[0].(*[]string); ok {
			*p = append(*p, "B")
		}
	}
}

func TestHooksRemoveAction(t *testing.T) {
	h := wp.NewHooks()
	var got []string
	h.AddAction("x", namedActionA, 10, 1)
	h.AddAction("x", namedActionB, 10, 1)
	if !h.RemoveAction("x", namedActionA, 10) {
		t.Fatal("RemoveAction returned false")
	}
	if h.RemoveAction("x", namedActionA, 10) {
		t.Fatal("second RemoveAction should be false")
	}
	h.DoAction("x", &got)
	if fmt.Sprint(got) != "[B]" {
		t.Fatalf("got %v", got)
	}
	if !h.HasAction("x") {
		t.Fatal("expected HasAction true")
	}
	_ = h.RemoveAction("x", namedActionB, 10)
	if h.HasAction("x") {
		t.Fatal("expected HasAction false after remove all")
	}
}

func TestHooksNestedDoAction(t *testing.T) {
	h := wp.NewHooks()
	var got []string
	h.AddAction("outer", func(args ...any) {
		got = append(got, "outer-start")
		if h.CurrentAction() != "outer" {
			t.Errorf("CurrentAction=%q want outer", h.CurrentAction())
		}
		if !h.DoingAction("outer") {
			t.Error("DoingAction(outer) false")
		}
		h.DoAction("inner")
		got = append(got, "outer-end")
	}, 10, 0)
	h.AddAction("inner", func(args ...any) {
		got = append(got, "inner")
		if h.CurrentAction() != "inner" {
			t.Errorf("CurrentAction=%q want inner", h.CurrentAction())
		}
		if !h.DoingAction("outer") || !h.DoingAction("inner") {
			t.Error("nested DoingAction stack incomplete")
		}
	}, 10, 0)
	h.DoAction("outer")
	if fmt.Sprint(got) != "[outer-start inner outer-end]" {
		t.Fatalf("got %v", got)
	}
	if h.CurrentAction() != "" || h.DoingAction("") {
		t.Fatal("stack not cleared")
	}
}

func TestHooksFilterChaining(t *testing.T) {
	h := wp.NewHooks()
	h.AddFilter("title", func(value any, args ...any) any {
		return value.(string) + "-a"
	}, 10, 1)
	h.AddFilter("title", func(value any, args ...any) any {
		return value.(string) + "-b"
	}, 10, 1)
	h.AddFilter("title", func(value any, args ...any) any {
		return value.(string) + "-c"
	}, 20, 1)
	got := h.ApplyFilters("title", "x")
	if got != "x-a-b-c" {
		t.Fatalf("got %v", got)
	}
}

func namedFilterUpper(value any, args ...any) any {
	return value.(string) + "-U"
}

func TestHooksRemoveFilterAndAcceptedArgs(t *testing.T) {
	h := wp.NewHooks()
	h.AddFilter("t", namedFilterUpper, 10, 1)
	h.AddFilter("t", func(value any, args ...any) any {
		if len(args) != 1 {
			t.Errorf("acceptedArgs: got %d extra args", len(args))
		}
		return fmt.Sprintf("%v/%v", value, args[0])
	}, 11, 2)
	got := h.ApplyFilters("t", "hi", "extra", "ignored")
	if got != "hi-U/extra" {
		t.Fatalf("got %v", got)
	}
	if !h.RemoveFilter("t", namedFilterUpper, 10) {
		t.Fatal("RemoveFilter failed")
	}
	got = h.ApplyFilters("t", "hi", "extra")
	if got != "hi/extra" {
		t.Fatalf("after remove got %v", got)
	}
	if !h.HasFilter("t") {
		t.Fatal("HasFilter false")
	}
}

func TestHooksCurrentFilterStack(t *testing.T) {
	h := wp.NewHooks()
	h.AddFilter("f1", func(value any, args ...any) any {
		if h.CurrentFilter() != "f1" {
			t.Errorf("CurrentFilter=%q", h.CurrentFilter())
		}
		_ = h.ApplyFilters("f2", value)
		return value
	}, 10, 1)
	h.AddFilter("f2", func(value any, args ...any) any {
		if h.CurrentFilter() != "f2" {
			t.Errorf("CurrentFilter=%q", h.CurrentFilter())
		}
		if !h.DoingFilter("f1") {
			t.Error("expected f1 on stack")
		}
		return value
	}, 10, 1)
	_ = h.ApplyFilters("f1", "v")
	if h.CurrentFilter() != "" {
		t.Fatal("stack not empty")
	}
}

func TestContextHoldsHooks(t *testing.T) {
	c := wp.NewContext(nil)
	if c.Hooks != nil {
		t.Fatal("Hooks should be nil until set")
	}
	h := wp.NewHooks()
	c.Hooks = h
	if c.Hooks != h {
		t.Fatal("Hooks not wired")
	}
	c.Hooks.AddAction("init", func(args ...any) {}, 10, 0)
	if !c.Hooks.HasAction("init") {
		t.Fatal("HasAction via Context")
	}
}

func TestHooksNilSafe(t *testing.T) {
	var h *wp.Hooks
	h.DoAction("x")
	if h.ApplyFilters("x", 1) != 1 {
		t.Fatal("nil ApplyFilters")
	}
	if h.HasAction("x") || h.HasFilter("x") {
		t.Fatal("nil Has*")
	}
	if h.CurrentFilter() != "" {
		t.Fatal("nil CurrentFilter")
	}
}

package wp_test

import (
	"context"
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
	c := wp.NewContext(context.TODO())
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
	h.DoActionRefArray("x", []any{1})
	if h.ApplyFilters("x", 1) != 1 {
		t.Fatal("nil ApplyFilters")
	}
	if h.ApplyFiltersRefArray("x", []any{1}) != 1 {
		t.Fatal("nil ApplyFiltersRefArray")
	}
	if h.DidAction("x") != 0 {
		t.Fatal("nil DidAction")
	}
	if h.HasAction("x") || h.HasFilter("x") {
		t.Fatal("nil Has*")
	}
	if h.CurrentFilter() != "" {
		t.Fatal("nil CurrentFilter")
	}
}

// P9.27: do_action_ref_array / apply_filters_ref_array / did_action.
func TestHooksDoActionRefArrayAndDidAction(t *testing.T) {
	h := wp.NewHooks()
	if h.DidAction("ref") != 0 {
		t.Fatal("DidAction before fire")
	}
	type bag struct{ N int }
	b := &bag{}
	h.AddAction("ref", func(args ...any) {
		if len(args) > 0 {
			if p, ok := args[0].(*bag); ok {
				p.N++
			}
		}
	}, 10, 1)
	h.DoActionRefArray("ref", []any{b, "extra"})
	if b.N != 1 {
		t.Fatalf("ref mutate N=%d want 1", b.N)
	}
	if h.DidAction("ref") != 1 {
		t.Fatalf("DidAction=%d want 1", h.DidAction("ref"))
	}
	h.DoAction("ref", b)
	if h.DidAction("ref") != 2 {
		t.Fatalf("DidAction after DoAction=%d want 2", h.DidAction("ref"))
	}
	// No callbacks still increments did_action (WP behavior).
	h.DoActionRefArray("never", nil)
	if h.DidAction("never") != 1 {
		t.Fatalf("DidAction(never)=%d want 1", h.DidAction("never"))
	}
}

func TestHooksApplyFiltersRefArray(t *testing.T) {
	h := wp.NewHooks()
	h.AddFilter("ref", func(value any, args ...any) any {
		s, _ := value.(string)
		extra := ""
		if len(args) > 0 {
			extra, _ = args[0].(string)
		}
		return s + "-" + extra
	}, 10, 2)
	got := h.ApplyFiltersRefArray("ref", []any{"hello", "world"})
	if got != "hello-world" {
		t.Fatalf("ApplyFiltersRefArray=%v", got)
	}
	if h.ApplyFiltersRefArray("ref", nil) != nil {
		t.Fatal("empty args should return nil")
	}
}

// P9.28: did_filter increments on ApplyFilters / ApplyFiltersRefArray.
func TestHooksDidFilter(t *testing.T) {
	h := wp.NewHooks()
	if h.DidFilter("f") != 0 {
		t.Fatal("DidFilter before apply")
	}
	_ = h.ApplyFilters("f", "x")
	if h.DidFilter("f") != 1 {
		t.Fatalf("DidFilter after ApplyFilters=%d want 1", h.DidFilter("f"))
	}
	_ = h.ApplyFiltersRefArray("f", []any{"y"})
	if h.DidFilter("f") != 2 {
		t.Fatalf("DidFilter after RefArray=%d want 2", h.DidFilter("f"))
	}
	// No callbacks still increments (WP $wp_filters).
	_ = h.ApplyFilters("never", 1)
	if h.DidFilter("never") != 1 {
		t.Fatalf("DidFilter(never)=%d want 1", h.DidFilter("never"))
	}
	// Empty RefArray does not bump.
	_ = h.ApplyFiltersRefArray("empty", nil)
	if h.DidFilter("empty") != 0 {
		t.Fatalf("empty RefArray should not bump DidFilter")
	}
	// Actions must not bump filter counts.
	h.DoAction("f")
	if h.DidFilter("f") != 2 {
		t.Fatalf("DoAction must not bump DidFilter, got %d", h.DidFilter("f"))
	}
	if h.DidFilter("") != 0 {
		t.Fatal("nil-safe empty tag")
	}
	var nilH *wp.Hooks
	if nilH.DidFilter("f") != 0 {
		t.Fatal("nil DidFilter")
	}
}

// P9.29: has_filter / has_action with callback → priority.
func TestHooksHasFilterFuncPriority(t *testing.T) {
	h := wp.NewHooks()
	fn := func(value any, args ...any) any { return value }
	other := func(value any, args ...any) any { return value }

	if _, ok := h.HasFilterFunc("t", fn, -1); ok {
		t.Fatal("expected miss before add")
	}
	h.AddFilter("t", fn, 20, 1)
	pri, ok := h.HasFilterFunc("t", fn, -1)
	if !ok || pri != 20 {
		t.Fatalf("HasFilterFunc any: pri=%d ok=%v", pri, ok)
	}
	if _, ok := h.HasFilterFunc("t", fn, 10); ok {
		t.Fatal("exact priority 10 should miss")
	}
	if pri, ok := h.HasFilterFunc("t", fn, 20); !ok || pri != 20 {
		t.Fatalf("exact 20: pri=%d ok=%v", pri, ok)
	}
	if _, ok := h.HasFilterFunc("t", other, -1); ok {
		t.Fatal("other callback should miss")
	}
	if !h.HasFilter("t") {
		t.Fatal("HasFilter bool still true")
	}
}

func TestHooksHasActionFuncPriority(t *testing.T) {
	h := wp.NewHooks()
	fn := func(args ...any) {}
	h.AddAction("a", fn, 5, 0)
	pri, ok := h.HasActionFunc("a", fn, -1)
	if !ok || pri != 5 {
		t.Fatalf("HasActionFunc: pri=%d ok=%v", pri, ok)
	}
	if !h.HasAction("a") {
		t.Fatal("HasAction bool")
	}
	var nilH *wp.Hooks
	if _, ok := nilH.HasFilterFunc("x", func(value any, args ...any) any { return value }, -1); ok {
		t.Fatal("nil HasFilterFunc")
	}
}

// P9.30: remove_all_filters / remove_all_actions.
func TestHooksRemoveAllFilters(t *testing.T) {
	h := wp.NewHooks()
	h.AddFilter("t", func(value any, args ...any) any { return value.(string) + "a" }, 10, 1)
	h.AddFilter("t", func(value any, args ...any) any { return value.(string) + "b" }, 20, 1)
	if !h.RemoveAllFilters("t", 10) {
		t.Fatal("RemoveAllFilters should return true")
	}
	if got := h.ApplyFilters("t", "x"); got != "xb" {
		t.Fatalf("after remove pri 10: %v", got)
	}
	_ = h.RemoveAllFilters("t", -1)
	if h.HasFilter("t") {
		t.Fatal("expected empty after remove all")
	}
	if got := h.ApplyFilters("t", "z"); got != "z" {
		t.Fatalf("identity after clear: %v", got)
	}
}

func TestHooksRemoveAllActions(t *testing.T) {
	h := wp.NewHooks()
	n := 0
	h.AddAction("a", func(args ...any) { n++ }, 10, 0)
	h.AddAction("a", func(args ...any) { n += 10 }, 20, 0)
	_ = h.RemoveAllActions("a", -1)
	h.DoAction("a")
	if n != 0 {
		t.Fatalf("n=%d after remove_all_actions", n)
	}
	var nilH *wp.Hooks
	if !nilH.RemoveAllFilters("x", -1) {
		t.Fatal("nil RemoveAllFilters")
	}
}

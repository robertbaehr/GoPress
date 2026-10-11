package wp_test

import (
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// P9.39: DoActionRefArray / ApplyFiltersRefArray Context helpers.
func TestDoActionRefArrayP939(t *testing.T) {
	wp.DoActionRefArray(nil, "a", []any{1})
	c := &wp.Context{Hooks: wp.NewHooks()}
	wp.DoActionRefArray(c, "", []any{1})
	var saw []any
	c.Hooks.AddAction("a", func(args ...any) {
		saw = append([]any{}, args...)
	}, 10, 2)
	wp.DoActionRefArray(c, "a", []any{"x", 2})
	if len(saw) != 2 || saw[0] != "x" || saw[1] != 2 {
		t.Fatalf("saw=%v", saw)
	}
	if wp.DidAction(c, "a") != 1 {
		t.Fatalf("DidAction=%d", wp.DidAction(c, "a"))
	}
}

func TestApplyFiltersRefArrayP939(t *testing.T) {
	if got := wp.ApplyFiltersRefArray(nil, "f", []any{"seed"}); got != "seed" {
		t.Fatalf("nil context=%v", got)
	}
	if got := wp.ApplyFiltersRefArray(nil, "f", nil); got != nil {
		t.Fatalf("nil args=%v", got)
	}
	c := &wp.Context{Hooks: wp.NewHooks()}
	if got := wp.ApplyFiltersRefArray(c, "", []any{"seed"}); got != "seed" {
		t.Fatalf("empty tag=%v", got)
	}
	c.Hooks.AddFilter("f", func(value any, args ...any) any {
		s, _ := value.(string)
		extra, _ := args[0].(string)
		return s + "-" + extra
	}, 10, 2)
	got := wp.ApplyFiltersRefArray(c, "f", []any{"hello", "world"})
	if got != "hello-world" {
		t.Fatalf("got=%v", got)
	}
	if wp.DidFilter(c, "f") != 1 {
		t.Fatalf("DidFilter=%d", wp.DidFilter(c, "f"))
	}
}

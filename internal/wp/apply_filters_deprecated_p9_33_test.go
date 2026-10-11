package wp_test

import (
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// P9.33: apply_filters_deprecated with no callbacks returns args[0] untouched.
func TestApplyFiltersDeprecatedNoCallbacksP933(t *testing.T) {
	h := wp.NewHooks()
	var deprecated int
	h.AddAction("deprecated_hook_run", func(args ...any) { deprecated++ }, 10, 4)
	got := h.ApplyFiltersDeprecated("old_filter", []any{"seed", "extra"}, "4.9.0", "new_filter", "")
	if got != "seed" {
		t.Fatalf("got=%v", got)
	}
	if deprecated != 0 {
		t.Fatalf("deprecated_hook_run fired without callbacks: %d", deprecated)
	}
}

// P9.33: apply_filters_deprecated notifies then applies the filter.
func TestApplyFiltersDeprecatedRunsP933(t *testing.T) {
	h := wp.NewHooks()
	var seenHook, seenReplacement, seenVersion, seenMessage string
	h.AddAction("deprecated_hook_run", func(args ...any) {
		if len(args) >= 4 {
			seenHook, _ = args[0].(string)
			seenReplacement, _ = args[1].(string)
			seenVersion, _ = args[2].(string)
			seenMessage, _ = args[3].(string)
		}
	}, 10, 4)
	h.AddFilter("old_filter", func(value any, args ...any) any {
		s, _ := value.(string)
		extra := ""
		if len(args) > 0 {
			extra, _ = args[0].(string)
		}
		return s + "-" + extra + "-filtered"
	}, 10, 2)

	got := h.ApplyFiltersDeprecated("old_filter", []any{"seed", "x"}, "5.0.0", "new_filter", "use new")
	if got != "seed-x-filtered" {
		t.Fatalf("got=%v", got)
	}
	if seenHook != "old_filter" || seenReplacement != "new_filter" || seenVersion != "5.0.0" || seenMessage != "use new" {
		t.Fatalf("deprecated_hook_run args: %q %q %q %q", seenHook, seenReplacement, seenVersion, seenMessage)
	}
}

// P9.33: do_action_deprecated no-ops without callbacks; fires when present.
func TestDoActionDeprecatedP933(t *testing.T) {
	h := wp.NewHooks()
	var deprecated, action int
	h.AddAction("deprecated_hook_run", func(args ...any) { deprecated++ }, 10, 4)
	h.DoActionDeprecated("old_action", []any{"a"}, "3.0.0", "new_action", "")
	if deprecated != 0 || action != 0 {
		t.Fatalf("expected no-op without callbacks: deprecated=%d action=%d", deprecated, action)
	}
	h.AddAction("old_action", func(args ...any) { action++ }, 10, 1)
	h.DoActionDeprecated("old_action", []any{"a"}, "3.0.0", "new_action", "msg")
	if deprecated != 1 || action != 1 {
		t.Fatalf("deprecated=%d action=%d", deprecated, action)
	}
}

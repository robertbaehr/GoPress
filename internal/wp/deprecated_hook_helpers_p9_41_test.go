package wp_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// P9.41: Context helpers for apply_filters_deprecated / do_action_deprecated.
func TestApplyFiltersDeprecatedHelperP941(t *testing.T) {
	c := testBootQuery(t)
	wp.LastDeprecatedHookNotice = ""
	c.Hooks.AddFilter("old_filter_p941", func(value any, _ ...any) any {
		if s, ok := value.(string); ok {
			return s + "-ok"
		}
		return value
	}, 10, 1)
	got := wp.ApplyFiltersDeprecated(c, "old_filter_p941", []any{"seed"}, "6.0.0", "new_filter", "prefer new")
	if got != "seed-ok" {
		t.Fatalf("got=%v", got)
	}
	if !strings.Contains(wp.LastDeprecatedHookNotice, "old_filter_p941") {
		t.Fatalf("notice=%q", wp.LastDeprecatedHookNotice)
	}
	if wp.ApplyFiltersDeprecated(nil, "x", []any{"a"}, "1.0", "", "") != "a" {
		t.Fatal("nil context should return seed")
	}
}

// P9.41: DoActionDeprecated helper fires action + notice.
func TestDoActionDeprecatedHelperP941(t *testing.T) {
	c := testBootQuery(t)
	wp.LastDeprecatedHookNotice = ""
	var fired int
	c.Hooks.AddAction("old_action_p941", func(_ ...any) { fired++ }, 10, 0)
	wp.DoActionDeprecated(c, "old_action_p941", nil, "5.5.0", "new_action", "")
	if fired != 1 {
		t.Fatalf("fired=%d", fired)
	}
	if !strings.Contains(wp.LastDeprecatedHookNotice, "old_action_p941") {
		t.Fatalf("notice=%q", wp.LastDeprecatedHookNotice)
	}
	wp.DoActionDeprecated(nil, "old_action_p941", nil, "5.5.0", "", "") // no-op
}

// P9.41: DeprecatedHookNotify helper.
func TestDeprecatedHookNotifyP941(t *testing.T) {
	c := testBootQuery(t)
	wp.LastDeprecatedHookNotice = ""
	wp.DeprecatedHookNotify(c, "legacy_p941", "2.0.0", "replacement", "msg")
	if !strings.Contains(wp.LastDeprecatedHookNotice, "legacy_p941") ||
		!strings.Contains(wp.LastDeprecatedHookNotice, "replacement") {
		t.Fatalf("notice=%q", wp.LastDeprecatedHookNotice)
	}
	wp.DeprecatedHookNotify(nil, "x", "1.0", "", "")
	if wp.LastDeprecatedHookNotice != "" {
		t.Fatalf("nil clear=%q", wp.LastDeprecatedHookNotice)
	}
}

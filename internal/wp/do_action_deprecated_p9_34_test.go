package wp_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// P9.34: do_action_deprecated records notice; empty args still fire; filter can suppress.
func TestDoActionDeprecatedNoticeP934(t *testing.T) {
	h := wp.NewHooks()
	wp.LastDeprecatedHookNotice = ""
	var fired int
	h.AddAction("old_action", func(args ...any) { fired++ }, 10, 0)
	h.DoActionDeprecated("old_action", nil, "4.6.0", "new_action", "prefer new")
	if fired != 1 {
		t.Fatalf("empty args should still fire action: %d", fired)
	}
	if !strings.Contains(wp.LastDeprecatedHookNotice, "old_action") ||
		!strings.Contains(wp.LastDeprecatedHookNotice, "4.6.0") ||
		!strings.Contains(wp.LastDeprecatedHookNotice, "new_action") ||
		!strings.Contains(wp.LastDeprecatedHookNotice, "prefer new") {
		t.Fatalf("notice=%q", wp.LastDeprecatedHookNotice)
	}

	h.AddFilter("deprecated_hook_trigger_error", func(value any, _ ...any) any {
		return false
	}, 10, 1)
	h.DoActionDeprecated("old_action", []any{}, "4.6.0", "", "")
	if wp.LastDeprecatedHookNotice != "" {
		t.Fatalf("suppressed notice want empty, got %q", wp.LastDeprecatedHookNotice)
	}
}

func TestDeprecatedHookNoticeNoReplacementP934(t *testing.T) {
	got := wp.DeprecatedHookNotice("legacy_hook", "2.0.0", "", "gone")
	if !strings.Contains(got, "no alternative available") || !strings.Contains(got, "gone") {
		t.Fatalf("%q", got)
	}
}

// P9.34: apply_filters_deprecated also records LastDeprecatedHookNotice.
func TestApplyFiltersDeprecatedNoticeP934(t *testing.T) {
	h := wp.NewHooks()
	wp.LastDeprecatedHookNotice = ""
	h.AddFilter("old_filter", func(value any, _ ...any) any { return value }, 10, 1)
	_ = h.ApplyFiltersDeprecated("old_filter", []any{"v"}, "5.0.0", "new_filter", "")
	if !strings.Contains(wp.LastDeprecatedHookNotice, "old_filter") ||
		!strings.Contains(wp.LastDeprecatedHookNotice, "new_filter") {
		t.Fatalf("notice=%q", wp.LastDeprecatedHookNotice)
	}
}

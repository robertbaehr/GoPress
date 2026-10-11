package wp_test

import (
	"context"
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// P4.158: is_sticky filter + post_stuck / post_unstuck actions.
func TestIsStickyFilterAndStuckActionsP4158(t *testing.T) {
	cfg := testBootConfig()
	cfg.DBName = "gopress_go"
	c, err := wp.Boot(context.Background(), cfg)
	if err != nil {
		t.Skipf("Boot unavailable: %v", err)
	}
	const fixtureSticky = "a:1:{i:0;i:13;}"
	t.Cleanup(func() {
		_, _ = c.Options.UpdateOption("sticky_posts", fixtureSticky, "yes")
		_ = c.DB.Close()
	})
	_, _ = c.Options.UpdateOption("sticky_posts", "a:0:{}", "yes")

	var stuck, unstuck int64
	c.Hooks.AddAction("post_stuck", func(args ...any) {
		if len(args) > 0 {
			if id, ok := args[0].(int64); ok {
				stuck = id
			}
		}
	}, 10, 1)
	c.Hooks.AddAction("post_unstuck", func(args ...any) {
		if len(args) > 0 {
			if id, ok := args[0].(int64); ok {
				unstuck = id
			}
		}
	}, 10, 1)
	c.Hooks.AddFilter("is_sticky", func(value any, args ...any) any {
		// Force post 7 sticky via filter even when not in sticky_posts.
		if len(args) > 0 {
			if id, ok := args[0].(int64); ok && id == 7 {
				return true
			}
		}
		return value
	}, 10, 2)

	if !wp.IsSticky(c, 7) {
		t.Fatal("is_sticky filter should mark post 7 sticky")
	}
	if wp.IsSticky(c, 8) {
		t.Fatal("post 8 should not be sticky")
	}

	if err := wp.StickPost(c, 42); err != nil {
		t.Fatalf("StickPost: %v", err)
	}
	if stuck != 42 {
		t.Fatalf("post_stuck want 42 got %d", stuck)
	}
	if !wp.IsSticky(c, 42) {
		t.Fatal("expected sticky after StickPost")
	}
	// Duplicate stick should not re-fire.
	stuck = 0
	if err := wp.StickPost(c, 42); err != nil {
		t.Fatalf("StickPost again: %v", err)
	}
	if stuck != 0 {
		t.Fatalf("post_stuck should not re-fire on duplicate: %d", stuck)
	}

	if err := wp.UnstickPost(c, 42); err != nil {
		t.Fatalf("UnstickPost: %v", err)
	}
	if unstuck != 42 {
		t.Fatalf("post_unstuck want 42 got %d", unstuck)
	}
}

package wp_test

import (
	"context"
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestStickUnstickPost(t *testing.T) {
	cfg := testBootConfig()
	cfg.DBName = "gopress_go"
	c, err := wp.Boot(context.Background(), cfg)
	if err != nil {
		t.Skipf("Boot unavailable: %v", err)
	}
	t.Cleanup(func() { _ = c.DB.Close() })

	prev := ""
	if v := c.Options.GetOption("sticky_posts", nil); v != nil {
		prev = fmtSprint(v)
	}
	t.Cleanup(func() {
		if prev != "" {
			_, _ = c.Options.UpdateOption("sticky_posts", prev, "yes")
		} else {
			_, _ = c.Options.UpdateOption("sticky_posts", "a:0:{}", "yes")
		}
	})
	_, _ = c.Options.UpdateOption("sticky_posts", "a:0:{}", "yes")

	if wp.IsSticky(c, 99) {
		t.Fatal("expected not sticky")
	}
	if err := wp.StickPost(c, 99); err != nil {
		t.Fatalf("StickPost: %v", err)
	}
	if !wp.IsSticky(c, 99) {
		t.Fatal("expected sticky after StickPost")
	}
	if err := wp.StickPost(c, 99); err != nil {
		t.Fatalf("StickPost again: %v", err)
	}
	if err := wp.UnstickPost(c, 99); err != nil {
		t.Fatalf("UnstickPost: %v", err)
	}
	if wp.IsSticky(c, 99) {
		t.Fatal("expected unstuck")
	}
}

func fmtSprint(v any) string {
	switch t := v.(type) {
	case string:
		return t
	default:
		return ""
	}
}

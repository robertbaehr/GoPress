package wp_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// P3.156: GetDeletePostLink trash action + nonce for admin.
func TestGetDeletePostLinkP3156(t *testing.T) {
	c := testBootQuery(t)
	user, err := c.GetUserBy("login", "admin")
	if err != nil || user == nil {
		t.Fatalf("admin: %v", err)
	}
	c.CurrentUser = user

	got := wp.GetDeletePostLink(c, 1, false)
	if !strings.Contains(got, "post.php?") || !strings.Contains(got, "post=1") || !strings.Contains(got, "action=trash") {
		t.Fatalf("trash=%q", got)
	}
	if !strings.Contains(got, "_wpnonce=") {
		t.Fatalf("missing nonce: %q", got)
	}
	force := wp.GetDeletePostLink(c, 1, true)
	if !strings.Contains(force, "action=delete") {
		t.Fatalf("force=%q", force)
	}

	c.Hooks.AddFilter("get_delete_post_link", func(value any, _ ...any) any {
		return "https://filtered.example/delete"
	}, 10, 3)
	if got := wp.GetDeletePostLink(c, 1, false); got != "https://filtered.example/delete" {
		t.Fatalf("filtered=%q", got)
	}
}

// P3.156: anon denied.
func TestGetDeletePostLinkDeniedP3156(t *testing.T) {
	c := testBootQuery(t)
	c.CurrentUser = nil
	if got := wp.GetDeletePostLink(c, 1, false); got != "" {
		t.Fatalf("anon=%q", got)
	}
	if wp.EmptyTrashDays() != 30 {
		t.Fatalf("EMPTY_TRASH_DAYS=%d", wp.EmptyTrashDays())
	}
}

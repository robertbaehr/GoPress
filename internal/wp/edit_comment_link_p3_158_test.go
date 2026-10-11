package wp_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// P3.158: GetEditCommentLink for admin + edit_comment meta cap.
func TestGetEditCommentLinkP3158(t *testing.T) {
	c := testBootQuery(t)
	user, err := c.GetUserBy("login", "admin")
	if err != nil || user == nil {
		t.Fatalf("admin: %v", err)
	}
	c.CurrentUser = user

	got := wp.GetEditCommentLink(c, 1, "display")
	if !strings.Contains(got, "comment.php?") || !strings.Contains(got, "action=editcomment") || !strings.Contains(got, "c=1") {
		t.Fatalf("display=%q", got)
	}
	if !strings.Contains(got, "&amp;c=") {
		t.Fatalf("display should amp-escape: %q", got)
	}
	raw := wp.GetEditCommentLink(c, 1, "raw")
	if !strings.Contains(raw, "&c=") || strings.Contains(raw, "&amp;c=") {
		t.Fatalf("raw=%q", raw)
	}

	c.Hooks.AddFilter("get_edit_comment_link", func(value any, _ ...any) any {
		return "/filtered-edit-comment"
	}, 10, 3)
	if got := wp.GetEditCommentLink(c, 1, "display"); got != "/filtered-edit-comment" {
		t.Fatalf("filter=%q", got)
	}
}

// P3.158: anonymous / missing comment denied.
func TestGetEditCommentLinkDenyP3158(t *testing.T) {
	c := testBootQuery(t)
	if got := wp.GetEditCommentLink(c, 1, "display"); got != "" {
		t.Fatalf("anon=%q", got)
	}
	user, err := c.GetUserBy("login", "admin")
	if err != nil || user == nil {
		t.Fatalf("admin: %v", err)
	}
	c.CurrentUser = user
	if got := wp.GetEditCommentLink(c, 999999, "display"); got != "" {
		t.Fatalf("missing=%q", got)
	}
}

package wp_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// P3.155: GetEditPostLink for admin user; display uses &amp;action=edit.
func TestGetEditPostLinkP3155(t *testing.T) {
	c := testBootQuery(t)
	user, err := c.GetUserBy("login", "admin")
	if err != nil || user == nil {
		t.Fatalf("admin: %v", err)
	}
	c.CurrentUser = user

	got := wp.GetEditPostLink(c, 1, "display")
	if !strings.Contains(got, "post.php?post=1") || !strings.Contains(got, "&amp;action=edit") {
		t.Fatalf("display=%q", got)
	}
	plain := wp.GetEditPostLink(c, 1, "url")
	if !strings.Contains(plain, "&action=edit") || strings.Contains(plain, "&amp;") {
		t.Fatalf("url=%q", plain)
	}
	if wp.GetEditPostLink(c, 0, "") != "" || wp.GetEditPostLink(nil, 1, "") != "" {
		t.Fatal("empty guards")
	}

	c.Hooks.AddFilter("get_edit_post_link", func(value any, _ ...any) any {
		return "https://filtered.example/edit"
	}, 10, 3)
	if got := wp.GetEditPostLink(c, 1, "display"); got != "https://filtered.example/edit" {
		t.Fatalf("filtered=%q", got)
	}
}

// P3.155: anonymous user gets empty edit link.
func TestGetEditPostLinkDeniedP3155(t *testing.T) {
	c := testBootQuery(t)
	c.CurrentUser = nil
	if got := wp.GetEditPostLink(c, 1, "display"); got != "" {
		t.Fatalf("anon=%q", got)
	}
}

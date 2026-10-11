package wp_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// P3.161: GetEditTermLink for admin + edit_term→manage_categories.
func TestGetEditTermLinkP3161(t *testing.T) {
	c := testBootQuery(t)
	user, err := c.GetUserBy("login", "admin")
	if err != nil || user == nil {
		t.Fatalf("admin: %v", err)
	}
	c.CurrentUser = user

	got := wp.GetEditTermLink(c, 1, "category", "")
	if !strings.Contains(got, "term.php") || !strings.Contains(got, "taxonomy=category") || !strings.Contains(got, "tag_ID=1") {
		t.Fatalf("link=%q", got)
	}
	if !strings.Contains(got, "post_type=post") {
		t.Fatalf("post_type missing: %q", got)
	}

	c.Hooks.AddFilter("get_edit_term_link", func(value any, _ ...any) any {
		return "/filtered-edit-term"
	}, 10, 4)
	if got := wp.GetEditTermLink(c, 1, "category", ""); got != "/filtered-edit-term" {
		t.Fatalf("filter=%q", got)
	}
}

// P3.161: anonymous / missing term denied.
func TestGetEditTermLinkDenyP3161(t *testing.T) {
	c := testBootQuery(t)
	if got := wp.GetEditTermLink(c, 1, "category", ""); got != "" {
		t.Fatalf("anon=%q", got)
	}
	user, err := c.GetUserBy("login", "admin")
	if err != nil || user == nil {
		t.Fatalf("admin: %v", err)
	}
	c.CurrentUser = user
	if got := wp.GetEditTermLink(c, 999999, "category", ""); got != "" {
		t.Fatalf("missing=%q", got)
	}
}

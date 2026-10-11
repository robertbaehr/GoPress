package wp_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// P3.159: GetEditUserLink self → profile.php; filter; anon deny.
func TestGetEditUserLinkP3159(t *testing.T) {
	c := testBootQuery(t)
	user, err := c.GetUserBy("login", "admin")
	if err != nil || user == nil {
		t.Fatalf("admin: %v", err)
	}
	c.CurrentUser = user

	got := wp.GetEditUserLink(c, 0)
	if !strings.Contains(got, "profile.php") {
		t.Fatalf("self link=%q", got)
	}
	got = wp.GetEditUserLink(c, user.ID)
	if !strings.Contains(got, "profile.php") {
		t.Fatalf("self by id=%q", got)
	}

	c.Hooks.AddFilter("get_edit_user_link", func(value any, _ ...any) any {
		return "/filtered-edit-user"
	}, 10, 2)
	if got := wp.GetEditUserLink(c, user.ID); got != "/filtered-edit-user" {
		t.Fatalf("filter=%q", got)
	}
}

// P3.159: anonymous denied.
func TestGetEditUserLinkDenyP3159(t *testing.T) {
	c := testBootQuery(t)
	if got := wp.GetEditUserLink(c, 1); got != "" {
		t.Fatalf("anon=%q", got)
	}
}

package wp_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestGetBodyClassLoggedInAndFilter(t *testing.T) {
	c := testBootQuery(t)
	if cls := wp.GetBodyClass(c, "home blog"); !strings.Contains(cls, "home") || strings.Contains(cls, "logged-in") {
		t.Fatalf("anon classes=%q", cls)
	}

	admin, err := c.GetUserBy("login", "admin")
	if err != nil || admin == nil {
		t.Fatalf("admin: %v", err)
	}
	c.CurrentUser = admin
	cls := wp.GetBodyClass(c, "home blog")
	for _, want := range []string{"home", "blog", "logged-in", "admin-bar", "no-customize-support"} {
		if !strings.Contains(" "+cls+" ", " "+want+" ") {
			t.Fatalf("missing %q in %q", want, cls)
		}
	}

	c.Hooks.AddFilter("body_class", func(value any, args ...any) any {
		parts, _ := value.([]string)
		return append(parts, "custom-from-filter")
	}, 10, 2)
	cls = wp.GetBodyClass(c, "home")
	if !strings.Contains(cls, "custom-from-filter") {
		t.Fatalf("filter not applied: %q", cls)
	}
}

func TestBodyClassWithAdminBarCompat(t *testing.T) {
	c := testBootQuery(t)
	admin, err := c.GetUserBy("login", "admin")
	if err != nil || admin == nil {
		t.Fatalf("admin: %v", err)
	}
	c.CurrentUser = admin
	if cls := wp.BodyClassWithAdminBar(c, "home"); !strings.Contains(cls, "admin-bar") || !strings.Contains(cls, "logged-in") {
		t.Fatalf("got %q", cls)
	}
}

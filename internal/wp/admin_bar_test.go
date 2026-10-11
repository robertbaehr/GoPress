package wp_test

import (
	"context"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestAdminBarHTMLAnonymousEmpty(t *testing.T) {
	c := wp.NewContext(context.Background())
	if got := wp.AdminBarHTML(c); got != "" {
		t.Fatalf("got %q", got)
	}
}

func TestAdminBarHTMLLoggedIn(t *testing.T) {
	c := testBootQuery(t)
	user, err := c.GetUserBy("login", "admin")
	if err != nil || user == nil {
		t.Fatalf("admin: %v", err)
	}
	c.CurrentUser = user
	html := wp.AdminBarHTML(c)
	if !strings.Contains(html, `id="wpadminbar"`) {
		t.Fatalf("missing wpadminbar: %q", html)
	}
	if !strings.Contains(html, "Howdy, admin") {
		t.Fatalf("missing howdy: %q", html)
	}
	if !strings.Contains(html, `/wp-admin/`) {
		t.Fatalf("missing dashboard link: %q", html)
	}
	if !strings.Contains(html, `id="adminbarsearch"`) {
		t.Fatalf("missing search form: %q", html)
	}
	if !strings.Contains(html, `id="wp-admin-bar-comments"`) {
		t.Fatalf("missing comments: %q", html)
	}
	cls := wp.BodyClassWithAdminBar(c, "home")
	for _, want := range []string{"home", "logged-in", "admin-bar", "no-customize-support"} {
		if !strings.Contains(" "+cls+" ", " "+want+" ") {
			t.Fatalf("missing %q in %q", want, cls)
		}
	}
}

func TestIsAdminBarShowingFilter(t *testing.T) {
	c := testBootQuery(t)
	user, err := c.GetUserBy("login", "admin")
	if err != nil || user == nil {
		t.Fatalf("admin: %v", err)
	}
	c.CurrentUser = user
	if !wp.IsAdminBarShowing(c) {
		t.Fatal("expected admin bar showing for logged-in admin")
	}
	c.Hooks.AddFilter("show_admin_bar", func(value any, _ ...any) any {
		return false
	}, 10, 1)
	if wp.IsAdminBarShowing(c) {
		t.Fatal("filter should hide admin bar")
	}
	if got := wp.AdminBarHTML(c); got != "" {
		t.Fatalf("hidden bar should be empty: %q", got)
	}
}

func TestIsAdminBarShowingAjaxHidden(t *testing.T) {
	c := testBootQuery(t)
	user, err := c.GetUserBy("login", "admin")
	if err != nil || user == nil {
		t.Fatalf("admin: %v", err)
	}
	c.CurrentUser = user
	c.Scope.SetAjax(true)
	if wp.IsAdminBarShowing(c) {
		t.Fatal("ajax should hide admin bar")
	}
}

func TestAdminBarEditLinkSingular(t *testing.T) {
	c := testBootQuery(t)
	user, err := c.GetUserBy("login", "admin")
	if err != nil || user == nil {
		t.Fatalf("admin: %v", err)
	}
	c.CurrentUser = user
	c.Scope.IsSingular = true
	c.Scope.QueriedObjectID = 1
	html := wp.AdminBarHTML(c)
	if !strings.Contains(html, `id="wp-admin-bar-edit"`) {
		t.Fatalf("missing edit node: %q", html)
	}
	if !strings.Contains(html, "post.php?post=1&amp;action=edit") {
		t.Fatalf("missing edit href: %q", html)
	}
}

func TestShowAdminBarSetter(t *testing.T) {
	c := testBootQuery(t)
	user, err := c.GetUserBy("login", "admin")
	if err != nil || user == nil {
		t.Fatalf("admin: %v", err)
	}
	c.CurrentUser = user
	wp.ShowAdminBar(c, false)
	if wp.IsAdminBarShowing(c) {
		t.Fatal("ShowAdminBar(false) should hide")
	}
	wp.ShowAdminBar(c, true)
	if !wp.IsAdminBarShowing(c) {
		t.Fatal("ShowAdminBar(true) should show")
	}
}

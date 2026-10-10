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
	if cls := wp.BodyClassWithAdminBar(c, "home"); cls != "home admin-bar" {
		t.Fatalf("class=%q", cls)
	}
}

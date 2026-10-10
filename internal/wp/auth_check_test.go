package wp_test

import (
	"context"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestAuthCheckLoggedOut(t *testing.T) {
	c, err := wp.Boot(context.Background(), testBootConfig())
	if err != nil {
		t.Skip(err)
	}
	defer c.DB.Close()
	if c.AuthCheck() {
		t.Fatal("expected AuthCheck false when logged out")
	}
}

func TestAuthCheckLoggedIn(t *testing.T) {
	c, err := wp.Boot(context.Background(), testBootConfig())
	if err != nil {
		t.Skip(err)
	}
	defer c.DB.Close()
	user, err := c.GetUserBy("login", "admin")
	if err != nil || user == nil {
		t.Fatalf("admin: %v", err)
	}
	c.CurrentUser = user
	if !c.AuthCheck() {
		t.Fatal("expected AuthCheck true when logged in")
	}
}

func TestAuthCheckHTML(t *testing.T) {
	c, err := wp.Boot(context.Background(), testBootConfig())
	if err != nil {
		t.Skip(err)
	}
	defer c.DB.Close()
	html := c.AuthCheckHTML()
	if !strings.Contains(html, `id="wp-auth-check-wrap"`) {
		t.Fatalf("missing wrap: %q", html)
	}
	if !strings.Contains(html, "interim-login=1") {
		t.Fatalf("missing interim-login: %q", html)
	}
}

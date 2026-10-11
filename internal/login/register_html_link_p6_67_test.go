package login_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/login"
)

// P6.67: register filter replaces Register nav link when registration is open.
func TestRegisterHTMLLinkFilterP667(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()
	orig := c.Options.GetOption("users_can_register", 0)
	t.Cleanup(func() { _, _ = c.Options.UpdateOption("users_can_register", orig, "yes") })
	_, _ = c.Options.UpdateOption("users_can_register", 1, "yes")

	c.Hooks.AddFilter("register", func(value any, _ ...any) any {
		return `<a class="wp-login-register custom" href="/custom-reg">Sign up</a>`
	}, 10, 1)

	req := httptest.NewRequest(http.MethodGet, "/wp-login.php", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&login.Handler{}).Handle(rec, req, c)
	body := rec.Body.String()
	if !strings.Contains(body, `class="wp-login-register custom"`) || !strings.Contains(body, "Sign up") {
		t.Fatalf("filter not applied: %q", body[:min(900, len(body))])
	}
	if strings.Contains(body, ">Register</a>") {
		t.Fatal("default Register text should be replaced")
	}
}

// P6.67: default Register link when filter unset.
func TestRegisterHTMLLinkDefaultP667(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()
	orig := c.Options.GetOption("users_can_register", 0)
	t.Cleanup(func() { _, _ = c.Options.UpdateOption("users_can_register", orig, "yes") })
	_, _ = c.Options.UpdateOption("users_can_register", 1, "yes")

	req := httptest.NewRequest(http.MethodGet, "/wp-login.php", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&login.Handler{}).Handle(rec, req, c)
	body := rec.Body.String()
	if !strings.Contains(body, `class="wp-login-register"`) || !strings.Contains(body, ">Register</a>") {
		t.Fatalf("default Register missing: %q", body[:min(900, len(body))])
	}
}

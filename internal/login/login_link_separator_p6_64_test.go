package login_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/login"
)

// P6.64: login_link_separator filter replaces nav separator; esc_html applied.
func TestLoginLinkSeparatorFilterP664(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()
	orig := c.Options.GetOption("users_can_register", 0)
	_, _ = c.Options.UpdateOption("users_can_register", "1", "yes")
	t.Cleanup(func() { _, _ = c.Options.UpdateOption("users_can_register", orig, "yes") })

	c.Hooks.AddFilter("login_link_separator", func(value any, _ ...any) any {
		return " :: "
	}, 10, 1)

	req := httptest.NewRequest(http.MethodGet, "/wp-login.php", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&login.Handler{}).Handle(rec, req, c)
	body := rec.Body.String()
	if !strings.Contains(body, `class="wp-login-register"`) {
		t.Fatalf("expected register link when registration open: %q", body[strings.Index(body, `id="nav"`):min(len(body), strings.Index(body, `id="nav"`)+400)])
	}
	if !strings.Contains(body, " :: ") {
		t.Fatalf("expected filtered separator: %q", body[strings.Index(body, `id="nav"`):min(len(body), strings.Index(body, `id="nav"`)+400)])
	}
	if strings.Contains(body, " | ") && strings.Count(body, " :: ") == 0 {
		t.Fatal("default separator should be replaced")
	}
}

// P6.64: separator HTML is escaped.
func TestLoginLinkSeparatorEscapedP664(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()
	orig := c.Options.GetOption("users_can_register", 0)
	_, _ = c.Options.UpdateOption("users_can_register", "1", "yes")
	t.Cleanup(func() { _, _ = c.Options.UpdateOption("users_can_register", orig, "yes") })

	c.Hooks.AddFilter("login_link_separator", func(value any, _ ...any) any {
		return ` <b>x</b> `
	}, 10, 1)

	req := httptest.NewRequest(http.MethodGet, "/wp-login.php?action=register", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&login.Handler{}).Handle(rec, req, c)
	body := rec.Body.String()
	if !strings.Contains(body, " &lt;b&gt;x&lt;/b&gt; ") {
		t.Fatalf("separator should be esc_html'd: %q", body[strings.Index(body, `id="nav"`):min(len(body), strings.Index(body, `id="nav"`)+500)])
	}
	if strings.Contains(body, " <b>x</b> ") {
		t.Fatal("raw HTML separator must not appear")
	}
}

// P6.64: default separator on register form when filter unset.
func TestLoginLinkSeparatorDefaultP664(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()
	orig := c.Options.GetOption("users_can_register", 0)
	_, _ = c.Options.UpdateOption("users_can_register", "1", "yes")
	t.Cleanup(func() { _, _ = c.Options.UpdateOption("users_can_register", orig, "yes") })

	req := httptest.NewRequest(http.MethodGet, "/wp-login.php?action=register", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&login.Handler{}).Handle(rec, req, c)
	body := rec.Body.String()
	if !strings.Contains(body, " | ") {
		t.Fatalf("default separator missing: %q", body[strings.Index(body, `id="nav"`):min(len(body), strings.Index(body, `id="nav"`)+400)])
	}
}

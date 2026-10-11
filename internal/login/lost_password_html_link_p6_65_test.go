package login_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/login"
)

// P6.65: lost_password_html_link filter replaces the lost-password nav link.
func TestLostPasswordHTMLLinkFilterP665(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	c.Hooks.AddFilter("lost_password_html_link", func(value any, _ ...any) any {
		return `<a class="wp-login-lost-password custom" href="/custom-lost">Reset it</a>`
	}, 10, 1)

	req := httptest.NewRequest(http.MethodGet, "/wp-login.php", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&login.Handler{}).Handle(rec, req, c)
	body := rec.Body.String()
	if !strings.Contains(body, `class="wp-login-lost-password custom"`) || !strings.Contains(body, "Reset it") {
		t.Fatalf("filter not applied: %q", body[strings.Index(body, `id="nav"`):min(len(body), strings.Index(body, `id="nav"`)+400)])
	}
	if strings.Contains(body, "Lost your password?") {
		t.Fatal("default link text should be replaced")
	}
}

// P6.65: default lost-password link still present without filter.
func TestLostPasswordHTMLLinkDefaultP665(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	req := httptest.NewRequest(http.MethodGet, "/wp-login.php", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&login.Handler{}).Handle(rec, req, c)
	body := rec.Body.String()
	if !strings.Contains(body, `class="wp-login-lost-password"`) || !strings.Contains(body, "Lost your password?") {
		t.Fatalf("default link missing: %q", body[strings.Index(body, `id="nav"`):min(len(body), strings.Index(body, `id="nav"`)+400)])
	}
}

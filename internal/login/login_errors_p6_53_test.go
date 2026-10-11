package login_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/login"
)

func TestLoginErrorsFilterRewritesMarkup(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()
	c.Hooks.AddFilter("login_errors", func(value any, _ ...any) any {
		return `<p><strong>Error:</strong> filtered-login-error</p>`
	}, 10, 1)

	form := url.Values{}
	form.Set("log", "admin")
	form.Set("pwd", "wrong-password-xyz")
	form.Set("wp-submit", "Log In")
	form.Set("testcookie", "1")
	req := httptest.NewRequest(http.MethodPost, "/wp-login.php", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "wordpress_test_cookie", Value: "WP Cookie check"})
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&login.Handler{}).Handle(rec, req, c)
	body := rec.Body.String()
	if !strings.Contains(body, `id="login_error"`) {
		t.Fatalf("expected login_error: %q", body[:min(500, len(body))])
	}
	if !strings.Contains(body, "filtered-login-error") {
		t.Fatalf("login_errors filter not applied: %q", body[:min(800, len(body))])
	}
	if !strings.Contains(body, `class="notice notice-error"`) {
		t.Fatalf("expected notice-error chrome: %q", body[:min(800, len(body))])
	}
}

func TestLoginErrorNoticeEmptyFilterSuppresses(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()
	c.Hooks.AddFilter("login_errors", func(value any, _ ...any) any {
		return ""
	}, 10, 1)

	form := url.Values{}
	form.Set("log", "admin")
	form.Set("pwd", "wrong-password-xyz")
	form.Set("wp-submit", "Log In")
	form.Set("testcookie", "1")
	req := httptest.NewRequest(http.MethodPost, "/wp-login.php", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "wordpress_test_cookie", Value: "WP Cookie check"})
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&login.Handler{}).Handle(rec, req, c)
	body := rec.Body.String()
	if strings.Contains(body, `id="login_error"`) {
		t.Fatalf("empty login_errors should suppress notice: %q", body[:min(600, len(body))])
	}
}

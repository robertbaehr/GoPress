package login_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/login"
)

// P6.66: login_remember_me_help_text filter replaces Remember Me tooltip text.
func TestLoginRememberMeHelpTextP666(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()
	c.Hooks.AddFilter("login_remember_me_help_text", func(value any, _ ...any) any {
		return "CUSTOM-REMEMBER-HELP"
	}, 10, 1)

	req := httptest.NewRequest(http.MethodGet, "/wp-login.php", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&login.Handler{}).Handle(rec, req, c)
	body := rec.Body.String()
	if !strings.Contains(body, "CUSTOM-REMEMBER-HELP") {
		t.Fatalf("missing filtered help: %q", body[:min(900, len(body))])
	}
	if !strings.Contains(body, `id="rememberme-help-toggletip-text"`) {
		t.Fatal("missing toggletip text id")
	}
}

// P6.66: enable_login_autofocus=false omits wp_attempt_focus() call.
func TestEnableLoginAutofocusFilterP666(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()
	c.Hooks.AddFilter("enable_login_autofocus", func(value any, _ ...any) any {
		return false
	}, 10, 1)

	req := httptest.NewRequest(http.MethodGet, "/wp-login.php", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&login.Handler{}).Handle(rec, req, c)
	body := rec.Body.String()
	if !strings.Contains(body, "function wp_attempt_focus()") {
		t.Fatal("expected focus function definition")
	}
	if strings.Contains(body, "wp_attempt_focus();\n") {
		t.Fatal("autofocus disabled should not call wp_attempt_focus")
	}
}

// P6.66: login_error suppresses autofocus even when filter allows it.
func TestLoginAutofocusSkippedOnErrorP666(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()
	form := url.Values{}
	form.Set("log", "admin")
	form.Set("pwd", "wrong-password-p666")
	form.Set("wp-submit", "Log In")
	form.Set("testcookie", "1")
	req := httptest.NewRequest(http.MethodPost, "/wp-login.php", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&login.Handler{}).Handle(rec, req, c)
	body := rec.Body.String()
	if !strings.Contains(body, `id="login_error"`) {
		t.Fatalf("expected login_error: %q", body[:min(600, len(body))])
	}
	if strings.Contains(body, "wp_attempt_focus();\n") {
		t.Fatal("error form should not call wp_attempt_focus")
	}
}

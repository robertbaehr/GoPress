package login_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/login"
	"github.com/robertbaehr/GoPress/internal/wp"
)

// P6.70: incorrect_password fires wp_login_failed with username + *AuthError.
func TestWPLoginFailedIncorrectPasswordP670(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	var gotLogin string
	var gotCode string
	c.Hooks.AddAction("wp_login_failed", func(args ...any) {
		if len(args) != 2 {
			t.Fatalf("args=%v", args)
		}
		gotLogin, _ = args[0].(string)
		if ae, ok := args[1].(*wp.AuthError); ok && ae != nil {
			gotCode = ae.Code
		}
	}, 10, 2)

	form := url.Values{}
	form.Set("log", "admin")
	form.Set("pwd", "definitely-wrong-password")
	form.Set("wp-submit", "Log In")
	form.Set("testcookie", "1")
	req := httptest.NewRequest(http.MethodPost, "/wp-login.php", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "wordpress_test_cookie", Value: "WP Cookie check"})
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&login.Handler{}).Handle(rec, req, c)

	if gotLogin != "admin" {
		t.Fatalf("wp_login_failed login=%q", gotLogin)
	}
	if gotCode != "incorrect_password" {
		t.Fatalf("wp_login_failed code=%q", gotCode)
	}
}

// P6.70: empty_username does not fire wp_login_failed.
func TestWPLoginFailedIgnoresEmptyUsernameP670(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	fired := false
	c.Hooks.AddAction("wp_login_failed", func(_ ...any) {
		fired = true
	}, 10, 2)

	form := url.Values{}
	form.Set("log", "")
	form.Set("pwd", "x")
	form.Set("wp-submit", "Log In")
	form.Set("testcookie", "1")
	req := httptest.NewRequest(http.MethodPost, "/wp-login.php", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "wordpress_test_cookie", Value: "WP Cookie check"})
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&login.Handler{}).Handle(rec, req, c)

	if fired {
		t.Fatal("empty_username must not fire wp_login_failed")
	}
}

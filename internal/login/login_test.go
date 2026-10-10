package login_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/robertbaehr/GoPress/internal/config"
	"github.com/robertbaehr/GoPress/internal/login"
	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestLoginFormGET(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	req := httptest.NewRequest(http.MethodGet, "/wp-login.php", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&login.Handler{}).Handle(rec, req, c)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "wp-login.php") || !strings.Contains(body, `method="post"`) {
		t.Fatalf("expected login form, got %q", body)
	}
	if !strings.Contains(body, `name="log"`) || !strings.Contains(body, `name="pwd"`) {
		t.Fatalf("expected log/pwd fields: %q", body)
	}
	if !strings.Contains(body, `name="testcookie"`) || !strings.Contains(body, `user-pass-wrap`) {
		t.Fatalf("expected WP-shaped markup: %q", body)
	}
	if !strings.Contains(body, `login-action-login`) {
		t.Fatalf("expected login-action-login body class: %q", body)
	}
	if !strings.Contains(body, `wp-hide-pw`) || !strings.Contains(body, `admin-color-modern`) {
		t.Fatalf("expected WP login chrome (hide-pw + admin-color): %q", body)
	}
	if !strings.Contains(body, `load-styles.php`) || !strings.Contains(body, `name='robots'`) {
		t.Fatalf("expected WP login head assets: %q", body)
	}
	foundTestCookie := false
	for _, ck := range rec.Result().Cookies() {
		if ck.Name == "wordpress_test_cookie" {
			foundTestCookie = true
		}
	}
	if !foundTestCookie {
		t.Fatal("expected wordpress_test_cookie Set-Cookie")
	}
}

func TestLostPasswordFormGET(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	req := httptest.NewRequest(http.MethodGet, "/wp-login.php?action=lostpassword", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&login.Handler{}).Handle(rec, req, c)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `name="lostpasswordform"`) || !strings.Contains(body, `id="lostpasswordform"`) {
		t.Fatalf("expected lostpassword form: %q", body)
	}
	if !strings.Contains(body, `login-action-lostpassword`) {
		t.Fatalf("expected lostpassword body class: %q", body)
	}
	if strings.Contains(body, `name="loginform"`) || strings.Contains(body, `name="pwd"`) {
		t.Fatalf("lostpassword should not be generic login: %q", body)
	}
	if !strings.Contains(body, "Get New Password") {
		t.Fatalf("expected submit label: %q", body)
	}
}

func TestLoginPOSTSuccess(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	form := url.Values{}
	form.Set("log", "admin")
	form.Set("pwd", "password")
	form.Set("redirect_to", "/wp-admin/")
	req := httptest.NewRequest(http.MethodPost, "/wp-login.php", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&login.Handler{}).Handle(rec, req, c)

	if rec.Code != http.StatusFound {
		t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String())
	}
	if loc := rec.Header().Get("Location"); loc != "/wp-admin/" {
		t.Fatalf("Location=%q", loc)
	}
	cookies := rec.Result().Cookies()
	var loggedIn bool
	for _, ck := range cookies {
		if strings.HasPrefix(ck.Name, "wordpress_logged_in_") && ck.Value != "" {
			loggedIn = true
		}
	}
	if !loggedIn {
		t.Fatalf("expected logged_in cookie, got %#v", cookies)
	}
}

func TestLoginPOSTBadPassword(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	form := url.Values{}
	form.Set("log", "admin")
	form.Set("pwd", "wrong-password")
	req := httptest.NewRequest(http.MethodPost, "/wp-login.php", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&login.Handler{}).Handle(rec, req, c)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "Error") {
		t.Fatalf("expected error HTML, got %q", rec.Body.String())
	}
}

func TestLogoutClearsCookies(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	user, err := c.GetUserBy("login", "admin")
	if err != nil || user == nil {
		t.Fatalf("admin: %v", err)
	}
	set, err := c.SetAuthCookie(user, false, false, 0, "")
	if err != nil {
		t.Fatalf("cookie: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/wp-login.php?action=logout&redirect_to=/wp-admin/", nil)
	req.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&login.Handler{}).Handle(rec, req, c)

	if rec.Code != http.StatusFound {
		t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String())
	}
	loc := rec.Header().Get("Location")
	if !strings.Contains(loc, "loggedout=true") && !strings.Contains(loc, "wp-login.php") {
		t.Fatalf("Location=%q", loc)
	}
	cleared := map[string]bool{}
	for _, ck := range rec.Result().Cookies() {
		if ck.Name == set.LoggedInName || ck.Name == set.AuthName {
			if ck.MaxAge >= 0 && (ck.Expires.IsZero() || ck.Expires.After(time.Now())) {
				t.Fatalf("expected expired cookie %s: %#v", ck.Name, ck)
			}
			cleared[ck.Name] = true
		}
	}
	if !cleared[set.LoggedInName] {
		t.Fatalf("expected cleared logged_in cookie, got %#v", rec.Result().Cookies())
	}
}

func bootOrSkip(t *testing.T) *wp.Context {
	t.Helper()
	cfg := &config.Config{
		DBName: "gopress_go", DBUser: "gopress", DBPassword: "gopress",
		DBHost: "127.0.0.1:3306", DBCharset: "utf8mb4", TablePrefix: "wp_",
		ABSPATH:        "/workspace/",
		AuthKey:        "gopress-ref-auth-key-0001-fixed-for-oracle",
		SecureAuthKey:  "gopress-ref-secure-auth-key-0002-fixed",
		LoggedInKey:    "gopress-ref-logged-in-key-0003-fixed",
		NonceKey:       "gopress-ref-nonce-key-0004-fixed-oracle",
		AuthSalt:       "gopress-ref-auth-salt-0005-fixed-oracle",
		SecureAuthSalt: "gopress-ref-secure-auth-salt-0006-fixed",
		LoggedInSalt:   "gopress-ref-logged-in-salt-0007-fixed",
		NonceSalt:      "gopress-ref-nonce-salt-0008-fixed-oracle",
	}
	c, err := wp.Boot(context.Background(), cfg)
	if err != nil {
		t.Skipf("Boot unavailable: %v", err)
	}
	return c
}

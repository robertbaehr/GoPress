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
	// Fixture has no installed language packs — WP omits the switcher (P6.11).
	if strings.Contains(body, "language-switcher") {
		t.Fatalf("unexpected language-switcher without installed languages: %q", body)
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

func TestInterimLoginForm(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	req := httptest.NewRequest(http.MethodGet, "/wp-login.php?interim-login=1", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&login.Handler{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "interim-login") {
		t.Fatalf("expected interim-login body class: %q", body)
	}
	if !strings.Contains(body, `name="interim-login"`) {
		t.Fatalf("expected interim-login hidden field: %q", body)
	}
	if strings.Contains(body, `id="nav"`) || strings.Contains(body, `id="backtoblog"`) {
		t.Fatalf("interim login should omit nav/backtoblog: %q", body)
	}
}

func TestInterimLoginSuccess(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	form := url.Values{}
	form.Set("log", "admin")
	form.Set("pwd", "password")
	form.Set("interim-login", "1")
	form.Set("testcookie", "1")
	req := httptest.NewRequest(http.MethodPost, "/wp-login.php?interim-login=1", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&login.Handler{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, "interim-login-success") {
		t.Fatalf("expected interim-login-success: %q", body)
	}
	if !strings.Contains(body, "You have logged in successfully.") {
		t.Fatalf("expected success message: %q", body)
	}
	if loc := rec.Header().Get("Location"); loc != "" {
		t.Fatalf("interim success should not redirect, Location=%q", loc)
	}
}

func TestRegisterDisabledRedirect(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	req := httptest.NewRequest(http.MethodGet, "/wp-login.php?action=register", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&login.Handler{}).Handle(rec, req, c)

	if rec.Code != http.StatusFound {
		t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String())
	}
	loc := rec.Header().Get("Location")
	if !strings.Contains(loc, "registration=disabled") {
		t.Fatalf("Location=%q", loc)
	}
}

func TestRegistrationDisabledMessage(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	req := httptest.NewRequest(http.MethodGet, "/wp-login.php?registration=disabled", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&login.Handler{}).Handle(rec, req, c)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "User registration is currently not allowed.") {
		t.Fatalf("expected disabled message: %q", body)
	}
	if !strings.Contains(body, `id="login_error"`) {
		t.Fatalf("expected login_error: %q", body)
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

func TestLostPasswordPOSTGeneratesResetKey(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()
	if c.Mailer == nil {
		c.Mailer = wp.NewMailer()
	}
	c.Mailer.Clear()

	form := url.Values{}
	form.Set("user_login", "admin")
	form.Set("action", "lostpassword")
	req := httptest.NewRequest(http.MethodPost, "/wp-login.php?action=lostpassword", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&login.Handler{}).Handle(rec, req, c)

	if rec.Code != http.StatusFound {
		t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String())
	}
	loc := rec.Header().Get("Location")
	if !strings.Contains(loc, "checkemail=confirm") {
		t.Fatalf("Location=%q", loc)
	}
	if len(c.Mailer.Sent()) != 1 {
		t.Fatalf("expected 1 mail, got %d", len(c.Mailer.Sent()))
	}
	user, err := c.GetUserBy("login", "admin")
	if err != nil || user == nil {
		t.Fatalf("admin: %v", err)
	}
	if user.ActivationKey == "" || !strings.Contains(user.ActivationKey, "$generic$") {
		t.Fatalf("activation key not set: %q", user.ActivationKey)
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

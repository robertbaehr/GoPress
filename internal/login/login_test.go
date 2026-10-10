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

// TestLoginSetsIsLogin covers P2.68: wp-login.php marks Scope.IsLogin.
func TestLoginSetsIsLogin(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()
	req := httptest.NewRequest(http.MethodGet, "/wp-login.php", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&login.Handler{}).Handle(rec, req, c)
	if !c.Scope.IsLogin {
		t.Fatal("expected Scope.IsLogin after login Handle")
	}
	if !wp.IsLogin(c) {
		t.Fatal("expected IsLogin true after login Handle")
	}
}

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
	// P6.27: nocache_headers on login.
	if got := rec.Header().Get("Expires"); got != "Wed, 11 Jan 1984 05:00:00 GMT" {
		t.Fatalf("Expires=%q", got)
	}
	wantCC := "no-cache, must-revalidate, max-age=0, no-store, private"
	if got := rec.Header().Get("Cache-Control"); got != wantCC {
		t.Fatalf("Cache-Control=%q want %q", got, wantCC)
	}
}

func TestLoginNoCacheHeaders(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()
	req := httptest.NewRequest(http.MethodGet, "/wp-login.php?action=lostpassword", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&login.Handler{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d", rec.Code)
	}
	if got := rec.Header().Get("Cache-Control"); !strings.Contains(got, "no-store") {
		t.Fatalf("Cache-Control=%q", got)
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
	if !strings.Contains(body, `id="login-message"`) ||
		!strings.Contains(body, "Your session has expired. Please log in to continue where you left off.") {
		t.Fatalf("expected interim session-expired message: %q", body)
	}
	if strings.Contains(body, `id="login_error"`) {
		t.Fatalf("interim info message should not use login_error: %q", body)
	}
	if !strings.Contains(body, "html{background-color: transparent;}") {
		t.Fatalf("expected transparent interim chrome: %q", body)
	}
	// P6.22: interim form footer forces links to open in a new tab.
	if !strings.Contains(body, "links[i].target = '_blank'") {
		t.Fatalf("expected interim blank-target script: %q", body)
	}
}

func TestInterimLoginPostBodyOnly(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	// interim-login only in POST body (not query) — mirrors $_REQUEST in wp-login.php.
	form := url.Values{}
	form.Set("log", "admin")
	form.Set("pwd", "password")
	form.Set("interim-login", "1")
	form.Set("testcookie", "1")
	req := httptest.NewRequest(http.MethodPost, "/wp-login.php", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "wordpress_test_cookie", Value: "WP Cookie check"})
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&login.Handler{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, "interim-login-success") {
		t.Fatalf("expected interim success from POST-only flag: %q", body)
	}
	if loc := rec.Header().Get("Location"); loc != "" {
		t.Fatalf("interim success should not redirect, Location=%q", loc)
	}
}

// TestInterimLoginShakeOnBadPassword covers P6.29: interim auth failures shake.
func TestInterimLoginShakeOnBadPassword(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	form := url.Values{}
	form.Set("log", "admin")
	form.Set("pwd", "wrong-password")
	form.Set("interim-login", "1")
	form.Set("testcookie", "1")
	req := httptest.NewRequest(http.MethodPost, "/wp-login.php?interim-login=1", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "wordpress_test_cookie", Value: "WP Cookie check"})
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&login.Handler{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "interim-login") {
		t.Fatalf("expected interim-login body class on error: %q", body[:min(400, len(body))])
	}
	if strings.Contains(body, "interim-login-success") {
		t.Fatalf("bad password must not succeed interim login")
	}
	if !strings.Contains(body, `id="login_error"`) {
		t.Fatalf("expected login_error on interim failure: %q", body[:min(400, len(body))])
	}
	if !strings.Contains(body, "document.querySelector('form').classList.add('shake')") {
		t.Fatalf("expected interim login shake script: %q", body[:min(600, len(body))])
	}
	if !strings.Contains(body, `name="interim-login"`) {
		t.Fatalf("expected interim hidden field preserved: %q", body[:min(600, len(body))])
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
	req.AddCookie(&http.Cookie{Name: "wordpress_test_cookie", Value: "WP Cookie check"})
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&login.Handler{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, "interim-login") || !strings.Contains(body, "interim-login-success") {
		t.Fatalf("expected interim-login + interim-login-success: %q", body)
	}
	if !strings.Contains(body, "You have logged in successfully.") {
		t.Fatalf("expected success message: %q", body)
	}
	if !strings.Contains(body, "html{background-color: transparent;}") {
		t.Fatalf("expected transparent interim chrome: %q", body)
	}
	if !strings.Contains(body, `id="login-message"`) {
		t.Fatalf("expected login-message id: %q", body)
	}
	// P6.25: auth cookies must be set so the parent auth-check can proceed.
	cookies := rec.Result().Cookies()
	hasLoggedIn := false
	for _, ck := range cookies {
		if strings.Contains(ck.Name, "wordpress_logged_in") || strings.HasPrefix(ck.Name, "wordpress_logged_in_") {
			hasLoggedIn = true
			break
		}
	}
	if !hasLoggedIn {
		names := make([]string, 0, len(cookies))
		for _, ck := range cookies {
			names = append(names, ck.Name)
		}
		t.Fatalf("expected logged_in Set-Cookie on interim success, got %v", names)
	}
	if loc := rec.Header().Get("Location"); loc != "" {
		t.Fatalf("interim success should not redirect, Location=%q", loc)
	}
}

func TestInterimLoginSuccessRedirectTo(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	form := url.Values{}
	form.Set("log", "admin")
	form.Set("pwd", "password")
	form.Set("interim-login", "1")
	form.Set("redirect_to", "/wp-admin/")
	form.Set("testcookie", "1")
	req := httptest.NewRequest(http.MethodPost, "/wp-login.php?interim-login=1&redirect_to=%2Fwp-admin%2F", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "wordpress_test_cookie", Value: "WP Cookie check"})
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&login.Handler{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, "interim-login-success") {
		t.Fatalf("expected success class: %q", body)
	}
	if !strings.Contains(body, `href="/wp-admin/"`) || !strings.Contains(body, "Continue") {
		t.Fatalf("expected Continue link: %q", body)
	}
	if !strings.Contains(body, "window.top===window.self") {
		t.Fatalf("expected top-level redirect script: %q", body)
	}
	if loc := rec.Header().Get("Location"); loc != "" {
		t.Fatalf("must not HTTP-redirect (iframe auth-check): Location=%q", loc)
	}
}

func TestInterimLoginFormKeepsRedirectTo(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	req := httptest.NewRequest(http.MethodGet, "/wp-login.php?interim-login=1&redirect_to=%2Fwp-admin%2Fprofile.php", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&login.Handler{}).Handle(rec, req, c)
	body := rec.Body.String()
	if !strings.Contains(body, "interim-login=1") {
		t.Fatalf("expected interim action: %q", body)
	}
	if !strings.Contains(body, "redirect_to") || !strings.Contains(body, "profile.php") {
		t.Fatalf("expected redirect_to preserved on form: %q", body)
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
	body := rec.Body.String()
	if !strings.Contains(body, `id="login_error"`) {
		t.Fatalf("expected login_error: %q", body)
	}
	if !strings.Contains(body, "password you entered") || !strings.Contains(body, "<strong>admin</strong>") {
		t.Fatalf("expected incorrect_password message: %q", body)
	}
	if !strings.Contains(body, "action=lostpassword") {
		t.Fatalf("expected lost password link: %q", body)
	}
	// P6.28: incorrect_password is a shake_error_codes entry.
	if !strings.Contains(body, "document.querySelector('form').classList.add('shake')") {
		t.Fatalf("expected login shake script: %q", body)
	}
}

func TestLoginPOSTInvalidUsername(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	form := url.Values{}
	form.Set("log", "nosuchuser")
	form.Set("pwd", "x")
	req := httptest.NewRequest(http.MethodPost, "/wp-login.php", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&login.Handler{}).Handle(rec, req, c)
	body := rec.Body.String()
	if !strings.Contains(body, "not registered on this site") || !strings.Contains(body, "<strong>nosuchuser</strong>") {
		t.Fatalf("expected invalid_username: %q", body)
	}
}

func TestLoginPOSTEmptyFields(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	form := url.Values{}
	form.Set("log", "")
	form.Set("pwd", "")
	req := httptest.NewRequest(http.MethodPost, "/wp-login.php", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&login.Handler{}).Handle(rec, req, c)
	body := rec.Body.String()
	if !strings.Contains(body, `login-error-list`) {
		t.Fatalf("expected error list: %q", body)
	}
	if !strings.Contains(body, "username field is empty") || !strings.Contains(body, "password field is empty") {
		t.Fatalf("expected both empty messages: %q", body)
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

func TestResetPassFlow(t *testing.T) {
	c := bootOrSkip(t)
	t.Cleanup(func() { _ = c.DB.Close() })

	user, err := c.GetUserBy("login", "admin")
	if err != nil || user == nil {
		t.Fatalf("admin: %v", err)
	}
	// Restore before Close (Cleanup is LIFO).
	t.Cleanup(func() {
		if err := c.SetUserPassword(user.ID, "password"); err != nil {
			t.Errorf("restore admin password: %v", err)
		}
	})

	key, err := wp.GetPasswordResetKey(c, user)
	if err != nil {
		t.Fatal(err)
	}

	// Step 1: GET with key+login → redirect and set cookie.
	req := httptest.NewRequest(http.MethodGet, "/wp-login.php?action=rp&key="+key+"&login=admin", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&login.Handler{}).Handle(rec, req, c)
	if rec.Code != http.StatusFound {
		t.Fatalf("step1 status=%d body=%q", rec.Code, rec.Body.String())
	}
	var rpCookie *http.Cookie
	for _, ck := range rec.Result().Cookies() {
		if strings.HasPrefix(ck.Name, "wp-resetpass-") {
			rpCookie = ck
		}
	}
	if rpCookie == nil || !strings.Contains(rpCookie.Value, "admin:") {
		t.Fatalf("expected resetpass cookie, got %#v", rec.Result().Cookies())
	}

	// Step 2: GET with cookie → form.
	req = httptest.NewRequest(http.MethodGet, "/wp-login.php?action=rp", nil)
	req.AddCookie(rpCookie)
	rec = httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&login.Handler{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("step2 status=%d body=%q", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, `name="resetpassform"`) || !strings.Contains(body, `name="pass1"`) {
		t.Fatalf("expected resetpass form: %q", body)
	}
	if !strings.Contains(body, `name="rp_key"`) {
		t.Fatalf("expected rp_key hidden field: %q", body)
	}

	// Step 3: POST new password.
	newPass := "GoPress-Login-Reset-42"
	form := url.Values{}
	form.Set("pass1", newPass)
	form.Set("pass2", newPass)
	form.Set("rp_key", key)
	form.Set("action", "resetpass")
	req = httptest.NewRequest(http.MethodPost, "/wp-login.php?action=resetpass", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(rpCookie)
	rec = httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&login.Handler{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("step3 status=%d body=%q", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "Your password has been reset") {
		t.Fatalf("expected success message: %q", rec.Body.String())
	}

	reloaded, err := c.GetUserBy("login", "admin")
	if err != nil || reloaded == nil {
		t.Fatalf("reload: %v", err)
	}
	if !wp.CheckPassword(newPass, reloaded.Pass) {
		t.Fatal("password was not updated")
	}

	// Can log in with the new password.
	form = url.Values{}
	form.Set("log", "admin")
	form.Set("pwd", newPass)
	form.Set("redirect_to", "/wp-admin/")
	req = httptest.NewRequest(http.MethodPost, "/wp-login.php", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec = httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&login.Handler{}).Handle(rec, req, c)
	if rec.Code != http.StatusFound {
		t.Fatalf("login after reset status=%d body=%q", rec.Code, rec.Body.String())
	}
}

func TestResetPassInvalidKeyRedirects(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	req := httptest.NewRequest(http.MethodGet, "/wp-login.php?action=rp", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&login.Handler{}).Handle(rec, req, c)
	if rec.Code != http.StatusFound {
		t.Fatalf("status=%d", rec.Code)
	}
	loc := rec.Header().Get("Location")
	if !strings.Contains(loc, "action=lostpassword") || !strings.Contains(loc, "error=invalidkey") {
		t.Fatalf("Location=%q", loc)
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

	req := httptest.NewRequest(http.MethodGet, "/wp-login.php?action=logout", nil)
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

func TestLogoutHonorsRedirectTo(t *testing.T) {
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

	req := httptest.NewRequest(http.MethodGet, "/wp-login.php?action=logout&redirect_to=/sample-page/", nil)
	req.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&login.Handler{}).Handle(rec, req, c)

	if rec.Code != http.StatusFound {
		t.Fatalf("status=%d", rec.Code)
	}
	loc := rec.Header().Get("Location")
	if loc != "/sample-page/" {
		t.Fatalf("Location=%q want /sample-page/", loc)
	}
}

func TestLogoutRedirectToAdminHonored(t *testing.T) {
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
		t.Fatalf("status=%d", rec.Code)
	}
	loc := rec.Header().Get("Location")
	if loc != "/wp-admin/" {
		t.Fatalf("Location=%q want /wp-admin/ (P6.16 honors redirect_to)", loc)
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

func TestConfirmAdminEmailRequiresLogin(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	req := httptest.NewRequest(http.MethodGet, "/wp-login.php?action=confirm_admin_email&redirect_to=/wp-admin/", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&login.Handler{}).Handle(rec, req, c)
	if rec.Code != http.StatusFound {
		t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String())
	}
	loc := rec.Header().Get("Location")
	if !strings.Contains(loc, "wp-login.php") {
		t.Fatalf("Location=%q", loc)
	}
}

func TestConfirmAdminEmailForm(t *testing.T) {
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
	req := httptest.NewRequest(http.MethodGet, "/wp-login.php?action=confirm_admin_email&redirect_to=/wp-admin/", nil)
	req.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&login.Handler{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	for _, want := range []string{
		"admin-email-confirm-form",
		"Administration email verification",
		"correct-admin-email",
		"login-action-confirm_admin_email",
		"Remind me later",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing %q in %q", want, body)
		}
	}
}

func TestConfirmAdminEmailCorrectRedirect(t *testing.T) {
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
	// GET form to mint nonce under authenticated context.
	req := httptest.NewRequest(http.MethodGet, "/wp-login.php?action=confirm_admin_email&redirect_to=/wp-admin/", nil)
	req.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&login.Handler{}).Handle(rec, req, c)
	body := rec.Body.String()
	idx := strings.Index(body, `name="confirm_admin_email_nonce" value="`)
	if idx < 0 {
		t.Fatalf("missing nonce in %q", body)
	}
	idx += len(`name="confirm_admin_email_nonce" value="`)
	end := strings.Index(body[idx:], `"`)
	nonce := body[idx : idx+end]

	form := strings.NewReader("correct-admin-email=1&confirm_admin_email_nonce=" + nonce + "&redirect_to=%2Fwp-admin%2F")
	req = httptest.NewRequest(http.MethodPost, "/wp-login.php?action=confirm_admin_email", form)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	rec = httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&login.Handler{}).Handle(rec, req, c)
	if rec.Code != http.StatusFound {
		t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String())
	}
	loc := rec.Header().Get("Location")
	if !strings.Contains(loc, "/wp-admin") {
		t.Fatalf("Location=%q", loc)
	}
}

func TestPostPassSetsCookieAndRedirects(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	form := url.Values{}
	form.Set("post_password", "secret")
	form.Set("redirect_to", "http://127.0.0.1:8081/?p=1")
	req := httptest.NewRequest(http.MethodPost, "/wp-login.php?action=postpass", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&login.Handler{}).Handle(rec, req, c)
	if rec.Code != http.StatusFound {
		t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String())
	}
	if loc := rec.Header().Get("Location"); loc != "http://127.0.0.1:8081/?p=1" {
		t.Fatalf("Location=%q", loc)
	}
	wantName := "wp-postpass_" + c.CookieHash()
	found := false
	for _, ck := range rec.Result().Cookies() {
		if ck.Name == wantName {
			found = true
			if !strings.HasPrefix(ck.Value, "$P$") {
				t.Fatalf("cookie value=%q", ck.Value)
			}
			if ck.Path != "/" {
				t.Fatalf("path=%q", ck.Path)
			}
		}
	}
	if !found {
		t.Fatalf("missing %s cookie in %#v", wantName, rec.Result().Cookies())
	}
}

func TestPostPassMissingPasswordStillRedirects(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	form := url.Values{}
	form.Set("redirect_to", "/sample-page/")
	req := httptest.NewRequest(http.MethodPost, "/wp-login.php?action=postpass", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&login.Handler{}).Handle(rec, req, c)
	if rec.Code != http.StatusFound {
		t.Fatalf("status=%d", rec.Code)
	}
	if len(rec.Result().Cookies()) != 0 {
		for _, ck := range rec.Result().Cookies() {
			if strings.HasPrefix(ck.Name, "wp-postpass_") {
				t.Fatalf("should not set postpass cookie: %#v", ck)
			}
		}
	}
}

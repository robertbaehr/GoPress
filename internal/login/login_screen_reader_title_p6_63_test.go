package login_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/login"
)

// P6.63: login form emits screen-reader <h1> with the login_header title.
func TestLoginScreenReaderTitleP663(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	req := httptest.NewRequest(http.MethodGet, "/wp-login.php", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&login.Handler{}).Handle(rec, req, c)
	body := rec.Body.String()
	want := `<h1 class="screen-reader-text">Log In</h1>`
	if !strings.Contains(body, want) {
		t.Fatalf("expected screen-reader title: %q", body[:min(800, len(body))])
	}
	// Exactly one screen-reader h1 (centralized in writeLoginHead).
	if n := strings.Count(body, `class="screen-reader-text"`); n != 1 {
		t.Fatalf("screen-reader-text count=%d want 1", n)
	}
}

// P6.63: interim success uses empty title → no screen-reader <h1>.
func TestInterimSuccessOmitsScreenReaderTitleP663(t *testing.T) {
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
	body := rec.Body.String()
	if !strings.Contains(body, "interim-login-success") {
		t.Fatalf("expected interim success page: %q", body[:min(500, len(body))])
	}
	if strings.Contains(body, `class="screen-reader-text"`) {
		t.Fatalf("empty title must omit screen-reader h1: %q", body[:min(800, len(body))])
	}
}

// P6.63: confirm_admin_email skips screen-reader <h1> even with a non-empty title.
func TestConfirmAdminEmailOmitsScreenReaderTitleP663(t *testing.T) {
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
	body := rec.Body.String()
	if !strings.Contains(body, "login-action-confirm_admin_email") {
		t.Fatalf("expected confirm_admin_email page: %q", body[:min(500, len(body))])
	}
	if strings.Contains(body, `class="screen-reader-text"`) {
		t.Fatalf("confirm_admin_email must omit screen-reader h1: %q", body[:min(800, len(body))])
	}
	if !strings.Contains(body, "Confirm your administration email") {
		t.Fatalf("title still used in <title>: %q", body[:min(400, len(body))])
	}
}

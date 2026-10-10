package login_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/login"
)

func TestLogoutRejectsExternalRedirectTo(t *testing.T) {
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

	for _, evil := range []string{
		"https://evil.example/phish",
		"http://evil.example/",
		"//evil.example/path",
	} {
		req := httptest.NewRequest(http.MethodGet, "/wp-login.php?action=logout&redirect_to="+url.QueryEscape(evil), nil)
		req.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
		rec := httptest.NewRecorder()
		rc := c.ForRequest(rec, req)
		(&login.Handler{}).Handle(rec, req, rc)
		if rec.Code != http.StatusFound {
			t.Fatalf("status=%d for %q", rec.Code, evil)
		}
		loc := rec.Header().Get("Location")
		if loc != "/wp-admin/" {
			t.Fatalf("redirect_to=%q Location=%q want /wp-admin/", evil, loc)
		}
	}
}

func TestLoginPostRejectsExternalRedirectTo(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	form := url.Values{}
	form.Set("log", "admin")
	form.Set("pwd", "password")
	form.Set("redirect_to", "https://evil.example/steal")
	form.Set("wp-submit", "Log In")
	form.Set("testcookie", "1")

	req := httptest.NewRequest(http.MethodPost, "/wp-login.php", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	rc := c.ForRequest(rec, req)
	(&login.Handler{}).Handle(rec, req, rc)
	if rec.Code != http.StatusFound {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	loc := rec.Header().Get("Location")
	if loc != "/wp-admin/" {
		t.Fatalf("Location=%q want /wp-admin/ (external rejected)", loc)
	}
}

func TestLogoutAllowsSameHostAbsoluteRedirect(t *testing.T) {
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

	req := httptest.NewRequest(http.MethodGet, "/wp-login.php?action=logout&redirect_to="+url.QueryEscape("http://127.0.0.1:8082/sample-page/"), nil)
	req.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	rec := httptest.NewRecorder()
	rc := c.ForRequest(rec, req)
	(&login.Handler{}).Handle(rec, req, rc)
	if rec.Code != http.StatusFound {
		t.Fatalf("status=%d", rec.Code)
	}
	loc := rec.Header().Get("Location")
	if loc != "/sample-page/" {
		t.Fatalf("Location=%q want /sample-page/", loc)
	}
}

package login_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/login"
)

func TestLoginDefaultRedirectAdmin(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	form := url.Values{}
	form.Set("log", "admin")
	form.Set("pwd", "password")
	// Omit redirect_to — P6.34 defaults to admin.
	req := httptest.NewRequest(http.MethodPost, "/wp-login.php", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&login.Handler{}).Handle(rec, req, c)
	if rec.Code != http.StatusFound {
		t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String())
	}
	loc := rec.Header().Get("Location")
	if loc != "/wp-admin/" {
		t.Fatalf("Location=%q want /wp-admin/", loc)
	}
}

func TestLoginRedirectFilter(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	c.Hooks.AddFilter("login_redirect", func(value any, args ...any) any {
		return "/sample-page/"
	}, 10, 3)

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
		t.Fatalf("status=%d", rec.Code)
	}
	if loc := rec.Header().Get("Location"); loc != "/sample-page/" {
		t.Fatalf("Location=%q want /sample-page/ from login_redirect", loc)
	}
}

func TestLoginSubscriberDefaultRedirectProfile(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	sub, err := c.GetUserBy("login", "subscriber")
	if err != nil || sub == nil {
		t.Fatalf("subscriber: %v", err)
	}
	if err := c.SetUserPassword(sub.ID, "password"); err != nil {
		t.Fatal(err)
	}

	form := url.Values{}
	form.Set("log", "subscriber")
	form.Set("pwd", "password")
	req := httptest.NewRequest(http.MethodPost, "/wp-login.php", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&login.Handler{}).Handle(rec, req, c)
	if rec.Code != http.StatusFound {
		t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String())
	}
	loc := rec.Header().Get("Location")
	if !strings.Contains(loc, "profile.php") {
		t.Fatalf("subscriber Location=%q want profile.php", loc)
	}
	// Confirm cap expectation.
	if c.UserCan(sub, "edit_posts") {
		t.Fatal("subscriber should lack edit_posts")
	}
}

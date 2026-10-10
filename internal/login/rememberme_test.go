package login_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/robertbaehr/GoPress/internal/login"
)

func TestRememberMeLoginWritesLongLivedCookie(t *testing.T) {
	c := bootOrSkip(t)
	t.Cleanup(func() { _ = c.DB.Close() })

	form := url.Values{}
	form.Set("log", "admin")
	form.Set("pwd", "password")
	form.Set("rememberme", "forever")
	form.Set("redirect_to", "/wp-admin/")
	form.Set("testcookie", "1")
	req := httptest.NewRequest(http.MethodPost, "/wp-login.php", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "wordpress_test_cookie", Value: "WP Cookie check"})
	rec := httptest.NewRecorder()
	rc := c.ForRequest(rec, req)
	(&login.Handler{}).Handle(rec, req, rc)
	if rec.Code != http.StatusFound {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	found := false
	for _, ck := range rec.Result().Cookies() {
		if !strings.HasPrefix(ck.Name, "wordpress_logged_in_") {
			continue
		}
		found = true
		if ck.MaxAge < 13*24*3600 {
			t.Fatalf("logged_in MaxAge=%d", ck.MaxAge)
		}
		if ck.Expires.Before(time.Now().Add(13 * 24 * time.Hour)) {
			t.Fatalf("logged_in Expires=%v", ck.Expires)
		}
	}
	if !found {
		t.Fatal("missing wordpress_logged_in_ cookie")
	}
}

func TestRememberMeCheckedOnFailedLogin(t *testing.T) {
	c := bootOrSkip(t)
	t.Cleanup(func() { _ = c.DB.Close() })

	form := url.Values{}
	form.Set("log", "admin")
	form.Set("pwd", "wrong-password")
	form.Set("rememberme", "forever")
	form.Set("testcookie", "1")
	req := httptest.NewRequest(http.MethodPost, "/wp-login.php", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "wordpress_test_cookie", Value: "WP Cookie check"})
	rec := httptest.NewRecorder()
	rc := c.ForRequest(rec, req)
	(&login.Handler{}).Handle(rec, req, rc)
	body := rec.Body.String()
	if !strings.Contains(body, `id="rememberme"`) || !strings.Contains(body, `checked="checked"`) {
		t.Fatalf("expected rememberme checked on error form")
	}
}

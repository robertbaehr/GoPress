package login_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/login"
)

func TestLoginHTTPSSetsSecureAuthCookie(t *testing.T) {
	c := bootOrSkip(t)
	form := url.Values{}
	form.Set("log", "admin")
	form.Set("pwd", "password")
	form.Set("redirect_to", "/wp-admin/")
	form.Set("testcookie", "1")
	req := httptest.NewRequest(http.MethodPost, "/wp-login.php", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("X-Forwarded-Proto", "https")
	req.AddCookie(&http.Cookie{Name: "wordpress_test_cookie", Value: "WP Cookie check"})
	rec := httptest.NewRecorder()
	rc := c.ForRequest(rec, req)
	(&login.Handler{}).Handle(rec, req, rc)

	if rec.Code != http.StatusFound && rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var sawSec bool
	for _, ck := range rec.Result().Cookies() {
		if strings.HasPrefix(ck.Name, "wordpress_sec_") {
			sawSec = true
			if !ck.Secure {
				t.Fatalf("wordpress_sec_ cookie missing Secure flag")
			}
		}
	}
	if !sawSec {
		t.Fatalf("expected wordpress_sec_ cookie on HTTPS login; cookies=%v", cookieNames(rec))
	}
}

func cookieNames(rec *httptest.ResponseRecorder) []string {
	var names []string
	for _, ck := range rec.Result().Cookies() {
		names = append(names, ck.Name)
	}
	return names
}

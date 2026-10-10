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

func TestLostPasswordPreservesRedirectTo(t *testing.T) {
	c := bootOrSkip(t)
	t.Cleanup(func() { _ = c.DB.Close() })

	req := httptest.NewRequest(http.MethodGet, "/wp-login.php?action=lostpassword&redirect_to=%2Fsample-page%2F", nil)
	rec := httptest.NewRecorder()
	rc := c.ForRequest(rec, req)
	(&login.Handler{}).Handle(rec, req, rc)
	body := rec.Body.String()
	if !strings.Contains(body, `name="redirect_to"`) || !strings.Contains(body, `value="/sample-page/"`) {
		t.Fatalf("expected redirect_to preserved in form: %q", body)
	}
}

func TestLostPasswordPostHonorsRedirectTo(t *testing.T) {
	c := bootOrSkip(t)
	t.Cleanup(func() { _ = c.DB.Close() })
	if c.Mailer == nil {
		c.Mailer = wp.NewMailer()
	}
	c.Mailer.Clear()

	form := url.Values{}
	form.Set("user_login", "admin")
	form.Set("action", "lostpassword")
	form.Set("redirect_to", "/sample-page/")
	req := httptest.NewRequest(http.MethodPost, "/wp-login.php?action=lostpassword", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	rc := c.ForRequest(rec, req)
	(&login.Handler{}).Handle(rec, req, rc)
	if rec.Code != http.StatusFound {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	loc := rec.Header().Get("Location")
	if loc != "/sample-page/" {
		t.Fatalf("Location=%q want /sample-page/", loc)
	}
}

func TestLostPasswordPostRejectsExternalRedirect(t *testing.T) {
	c := bootOrSkip(t)
	t.Cleanup(func() { _ = c.DB.Close() })
	if c.Mailer == nil {
		c.Mailer = wp.NewMailer()
	}
	c.Mailer.Clear()

	form := url.Values{}
	form.Set("user_login", "admin")
	form.Set("action", "lostpassword")
	form.Set("redirect_to", "https://evil.example/steal")
	req := httptest.NewRequest(http.MethodPost, "/wp-login.php?action=lostpassword", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	rc := c.ForRequest(rec, req)
	(&login.Handler{}).Handle(rec, req, rc)
	if rec.Code != http.StatusFound {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	loc := rec.Header().Get("Location")
	if loc != "/wp-admin/" {
		t.Fatalf("Location=%q want /wp-admin/ fallback", loc)
	}
}

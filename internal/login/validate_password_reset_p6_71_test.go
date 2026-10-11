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

// P6.71: validate_password_reset fires with errors + user before reset.
func TestValidatePasswordResetActionP671(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	user, err := c.GetUserBy("login", "admin")
	if err != nil || user == nil {
		t.Fatalf("admin: %v", err)
	}
	t.Cleanup(func() {
		_ = c.SetUserPassword(user.ID, "password")
	})

	key, err := wp.GetPasswordResetKey(c, user)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodGet, "/wp-login.php?action=rp&key="+key+"&login=admin", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&login.Handler{}).Handle(rec, req, c)
	var rpCookie *http.Cookie
	for _, ck := range rec.Result().Cookies() {
		if strings.HasPrefix(ck.Name, "wp-resetpass-") {
			rpCookie = ck
		}
	}
	if rpCookie == nil {
		t.Fatal("expected resetpass cookie")
	}

	var fired int
	var gotUserID int64
	var gotErr string
	c.Hooks.AddAction("validate_password_reset", func(args ...any) {
		fired++
		if len(args) >= 1 {
			gotErr, _ = args[0].(string)
		}
		if len(args) >= 2 {
			if u, ok := args[1].(*wp.User); ok && u != nil {
				gotUserID = u.ID
			}
		}
	}, 10, 2)

	// GET form display still fires with empty errors.
	req = httptest.NewRequest(http.MethodGet, "/wp-login.php?action=rp", nil)
	req.AddCookie(rpCookie)
	rec = httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&login.Handler{}).Handle(rec, req, c)
	if fired != 1 || gotUserID != user.ID || gotErr != "" {
		t.Fatalf("GET fire=%d user=%d err=%q", fired, gotUserID, gotErr)
	}

	// Mismatch POST fires with error and does not reset.
	form := url.Values{}
	form.Set("pass1", "aaa")
	form.Set("pass2", "bbb")
	form.Set("rp_key", key)
	req = httptest.NewRequest(http.MethodPost, "/wp-login.php?action=resetpass", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(rpCookie)
	rec = httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&login.Handler{}).Handle(rec, req, c)
	if fired != 2 {
		t.Fatalf("POST fire count=%d", fired)
	}
	if gotErr == "" {
		t.Fatal("expected mismatch error string")
	}
	if !strings.Contains(rec.Body.String(), "do not match") {
		t.Fatalf("body=%q", rec.Body.String()[:min(400, len(rec.Body.String()))])
	}
}

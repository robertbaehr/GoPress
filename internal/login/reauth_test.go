package login_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/login"
)

func TestReauthForcesLoginForm(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	user, err := c.GetUserBy("login", "admin")
	if err != nil || user == nil {
		t.Skipf("admin unavailable: %v", err)
	}
	set, err := c.SetAuthCookie(user, false, false, 0, "")
	if err != nil {
		t.Fatalf("SetAuthCookie: %v", err)
	}

	// Without reauth, an authenticated visitor is redirected away from the form.
	req := httptest.NewRequest(http.MethodGet, "/wp-login.php?redirect_to=%2Fwp-admin%2F", nil)
	req.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	rec := httptest.NewRecorder()
	rc := c.ForRequest(rec, req)
	(&login.Handler{}).Handle(rec, req, rc)
	if rec.Code != http.StatusFound {
		t.Fatalf("expected redirect without reauth, status=%d body=%q", rec.Code, rec.Body.String())
	}
	if loc := rec.Header().Get("Location"); !strings.Contains(loc, "/wp-admin") {
		t.Fatalf("Location=%q", loc)
	}

	// reauth=1 forces the login form and clears auth cookies (P6.18).
	req = httptest.NewRequest(http.MethodGet, "/wp-login.php?reauth=1&redirect_to=%2Fwp-admin%2F", nil)
	req.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	rec = httptest.NewRecorder()
	rc = c.ForRequest(rec, req)
	(&login.Handler{}).Handle(rec, req, rc)
	if rec.Code != http.StatusOK {
		t.Fatalf("reauth status=%d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `id="loginform"`) {
		t.Fatalf("expected login form on reauth: %q", body)
	}
	cleared := false
	settingsCleared := false
	for _, ck := range rec.Result().Cookies() {
		if strings.HasPrefix(ck.Name, "wordpress_logged_in_") && (ck.MaxAge == 0 || ck.Value == " " || ck.Value == "") {
			cleared = true
		}
		if ck.Name == "wp-settings-1" || ck.Name == "wp-settings-time-1" {
			settingsCleared = true
		}
	}
	if !cleared {
		t.Fatalf("expected auth cookies cleared on reauth, cookies=%v", rec.Result().Cookies())
	}
	if !settingsCleared {
		t.Fatalf("expected wp-settings-{user} cleared, cookies=%v", rec.Result().Cookies())
	}
}

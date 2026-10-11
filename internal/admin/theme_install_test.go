package admin_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/admin"
)

func TestThemeInstallScreenStub(t *testing.T) {
	c := bootOrSkip(t)
	t.Cleanup(func() { _ = c.DB.Close() })
	user, err := c.GetUserBy("login", "admin")
	if err != nil || user == nil {
		t.Fatalf("admin: %v", err)
	}
	set, err := c.SetAuthCookie(user, false, false, 0, "")
	if err != nil {
		t.Fatalf("cookie: %v", err)
	}
	ck := &http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue}

	req := httptest.NewRequest(http.MethodGet, "/wp-admin/theme-install.php", nil)
	req.AddCookie(ck)
	rec := httptest.NewRecorder()
	rc := c.ForRequest(rec, req)
	if !(&admin.Handler{}).Handle(rec, req, rc, "/wp-admin/theme-install.php") {
		t.Fatal("theme-install.php not handled")
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if rec.Header().Get("X-GoPress-Theme-Install") != "disabled" {
		t.Fatalf("X-GoPress-Theme-Install=%q", rec.Header().Get("X-GoPress-Theme-Install"))
	}
	body := rec.Body.String()
	for _, want := range []string{
		"theme-install-php",
		"<h1>Add Themes</h1>",
		"gopress-theme-install-stub",
		"out of scope",
		"/wp-admin/themes.php",
		"/wp-admin/site-editor.php",
		"theme-browser",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing %q in %q", want, body)
		}
	}
}

func TestThemeInstallScreenRequiresLogin(t *testing.T) {
	c := bootOrSkip(t)
	t.Cleanup(func() { _ = c.DB.Close() })
	req := httptest.NewRequest(http.MethodGet, "/wp-admin/theme-install.php", nil)
	rec := httptest.NewRecorder()
	rc := c.ForRequest(rec, req)
	if !(&admin.Handler{}).Handle(rec, req, rc, "/wp-admin/theme-install.php") {
		t.Fatal("theme-install.php not handled")
	}
	if rec.Code != http.StatusFound {
		t.Fatalf("status=%d want redirect", rec.Code)
	}
	loc := rec.Header().Get("Location")
	if !strings.Contains(loc, "/wp-login.php") {
		t.Fatalf("Location=%q", loc)
	}
}

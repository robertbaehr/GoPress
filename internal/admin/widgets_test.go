package admin_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/admin"
)

func TestWidgetsScreenStub(t *testing.T) {
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

	req := httptest.NewRequest(http.MethodGet, "/wp-admin/widgets.php", nil)
	req.AddCookie(ck)
	rec := httptest.NewRecorder()
	rc := c.ForRequest(rec, req)
	if !(&admin.Handler{}).Handle(rec, req, rc, "/wp-admin/widgets.php") {
		t.Fatal("widgets.php not handled")
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if rec.Header().Get("X-GoPress-Widgets") != "disabled" {
		t.Fatalf("X-GoPress-Widgets=%q", rec.Header().Get("X-GoPress-Widgets"))
	}
	body := rec.Body.String()
	for _, want := range []string{
		"widgets-php",
		"<h1>Widgets</h1>",
		"gopress-widgets-stub",
		"unavailable",
		"/wp-admin/site-editor.php",
		"/wp-admin/themes.php",
		"blocks-widgets-container",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing %q in %q", want, body)
		}
	}
}

func TestWidgetsScreenRequiresLogin(t *testing.T) {
	c := bootOrSkip(t)
	t.Cleanup(func() { _ = c.DB.Close() })
	req := httptest.NewRequest(http.MethodGet, "/wp-admin/widgets.php", nil)
	rec := httptest.NewRecorder()
	rc := c.ForRequest(rec, req)
	if !(&admin.Handler{}).Handle(rec, req, rc, "/wp-admin/widgets.php") {
		t.Fatal("widgets.php not handled")
	}
	if rec.Code != http.StatusFound {
		t.Fatalf("status=%d want redirect", rec.Code)
	}
	loc := rec.Header().Get("Location")
	if !strings.Contains(loc, "/wp-login.php") {
		t.Fatalf("Location=%q", loc)
	}
}

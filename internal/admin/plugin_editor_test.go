package admin_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/admin"
)

func TestPluginEditorScreenStubForbidden(t *testing.T) {
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

	req := httptest.NewRequest(http.MethodGet, "/wp-admin/plugin-editor.php", nil)
	req.AddCookie(ck)
	rec := httptest.NewRecorder()
	rc := c.ForRequest(rec, req)
	if !(&admin.Handler{}).Handle(rec, req, rc, "/wp-admin/plugin-editor.php") {
		t.Fatal("plugin-editor.php not handled")
	}
	// edit_plugins is denied (oracle parity) → 403 stub with Plugins link.
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if rec.Header().Get("X-GoPress-Plugin-Editor") != "forbidden" {
		t.Fatalf("X-GoPress-Plugin-Editor=%q", rec.Header().Get("X-GoPress-Plugin-Editor"))
	}
	body := rec.Body.String()
	for _, want := range []string{
		"plugin-editor-php",
		"<h1>Plugin File Editor</h1>",
		"gopress-plugin-editor-stub",
		"not allowed to edit plugins",
		"out of scope",
		"/wp-admin/plugins.php",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing %q in %q", want, body)
		}
	}
}

func TestPluginEditorScreenRequiresLogin(t *testing.T) {
	c := bootOrSkip(t)
	t.Cleanup(func() { _ = c.DB.Close() })
	req := httptest.NewRequest(http.MethodGet, "/wp-admin/plugin-editor.php", nil)
	rec := httptest.NewRecorder()
	rc := c.ForRequest(rec, req)
	if !(&admin.Handler{}).Handle(rec, req, rc, "/wp-admin/plugin-editor.php") {
		t.Fatal("plugin-editor.php not handled")
	}
	if rec.Code != http.StatusFound {
		t.Fatalf("status=%d want redirect", rec.Code)
	}
	loc := rec.Header().Get("Location")
	if !strings.Contains(loc, "/wp-login.php") {
		t.Fatalf("Location=%q", loc)
	}
}

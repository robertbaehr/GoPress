package admin_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/admin"
)

func TestThemeEditorScreenStubForbidden(t *testing.T) {
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

	req := httptest.NewRequest(http.MethodGet, "/wp-admin/theme-editor.php", nil)
	req.AddCookie(ck)
	rec := httptest.NewRecorder()
	rc := c.ForRequest(rec, req)
	if !(&admin.Handler{}).Handle(rec, req, rc, "/wp-admin/theme-editor.php") {
		t.Fatal("theme-editor.php not handled")
	}
	// edit_themes is denied (oracle parity) → 403 stub with Site Editor link.
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if rec.Header().Get("X-GoPress-Theme-Editor") != "forbidden" {
		t.Fatalf("X-GoPress-Theme-Editor=%q", rec.Header().Get("X-GoPress-Theme-Editor"))
	}
	body := rec.Body.String()
	for _, want := range []string{
		"theme-editor-php",
		"<h1>Theme File Editor</h1>",
		"gopress-theme-editor-stub",
		"not allowed to edit templates",
		"out of scope",
		"/wp-admin/site-editor.php",
		"/wp-admin/themes.php",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing %q in %q", want, body)
		}
	}
}

func TestThemeEditorScreenRequiresLogin(t *testing.T) {
	c := bootOrSkip(t)
	t.Cleanup(func() { _ = c.DB.Close() })
	req := httptest.NewRequest(http.MethodGet, "/wp-admin/theme-editor.php", nil)
	rec := httptest.NewRecorder()
	rc := c.ForRequest(rec, req)
	if !(&admin.Handler{}).Handle(rec, req, rc, "/wp-admin/theme-editor.php") {
		t.Fatal("theme-editor.php not handled")
	}
	if rec.Code != http.StatusFound {
		t.Fatalf("status=%d want redirect", rec.Code)
	}
	loc := rec.Header().Get("Location")
	if !strings.Contains(loc, "/wp-login.php") {
		t.Fatalf("Location=%q", loc)
	}
}

// P7.156: theme/file/action chrome, pagenow, updated notice, theme header.
func TestThemeEditorDeepensQueryChromeP7156(t *testing.T) {
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

	req := httptest.NewRequest(http.MethodGet,
		"/wp-admin/theme-editor.php?theme=twentytwentyfour&file=style.css&action=edit&updated=true",
		nil)
	req.AddCookie(ck)
	rec := httptest.NewRecorder()
	rc := c.ForRequest(rec, req)
	if !(&admin.Handler{}).Handle(rec, req, rc, "/wp-admin/theme-editor.php") {
		t.Fatal("theme-editor.php not handled")
	}
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status=%d", rec.Code)
	}
	body := rec.Body.String()
	for _, want := range []string{
		`pagenow = "theme-editor"`,
		`data-theme="twentytwentyfour"`,
		`data-file="style.css"`,
		`data-action="edit"`,
		`File edited successfully`,
		`locale-en-us`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing %q in %q", want, body)
		}
	}
	if got := rec.Header().Get("X-GoPress-Theme-Editor-Theme"); got != "twentytwentyfour" {
		t.Fatalf("theme header=%q", got)
	}
}

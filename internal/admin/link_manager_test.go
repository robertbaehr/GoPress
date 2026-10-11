package admin_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/admin"
)

func TestLinkManagerScreenStubForbidden(t *testing.T) {
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

	for _, path := range []string{"/wp-admin/link-manager.php", "/wp-admin/link-add.php", "/wp-admin/link.php"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.AddCookie(ck)
		rec := httptest.NewRecorder()
		rc := c.ForRequest(rec, req)
		if !(&admin.Handler{}).Handle(rec, req, rc, path) {
			t.Fatalf("%s not handled", path)
		}
		if rec.Code != http.StatusForbidden {
			t.Fatalf("%s status=%d body=%s", path, rec.Code, rec.Body.String())
		}
		if rec.Header().Get("X-GoPress-Link-Manager") != "forbidden" {
			t.Fatalf("%s X-GoPress-Link-Manager=%q", path, rec.Header().Get("X-GoPress-Link-Manager"))
		}
		body := rec.Body.String()
		for _, want := range []string{
			"link-manager-php",
			"gopress-link-manager-stub",
			"not allowed to edit the links",
			"out of scope",
			"link_manager_enabled",
			"/wp-admin/",
		} {
			if !strings.Contains(body, want) {
				t.Fatalf("%s missing %q in %q", path, want, body)
			}
		}
	}
}

func TestLinkManagerScreenRequiresLogin(t *testing.T) {
	c := bootOrSkip(t)
	t.Cleanup(func() { _ = c.DB.Close() })
	req := httptest.NewRequest(http.MethodGet, "/wp-admin/link-manager.php", nil)
	rec := httptest.NewRecorder()
	rc := c.ForRequest(rec, req)
	if !(&admin.Handler{}).Handle(rec, req, rc, "/wp-admin/link-manager.php") {
		t.Fatal("link-manager.php not handled")
	}
	if rec.Code != http.StatusFound {
		t.Fatalf("status=%d want redirect", rec.Code)
	}
	loc := rec.Header().Get("Location")
	if !strings.Contains(loc, "/wp-login.php") {
		t.Fatalf("Location=%q", loc)
	}
}

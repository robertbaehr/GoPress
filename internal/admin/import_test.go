package admin_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/admin"
)

func TestImportStubScreen(t *testing.T) {
	c := bootOrSkip(t)
	user, err := c.GetUserBy("login", "admin")
	if err != nil || user == nil {
		t.Fatalf("admin: %v", err)
	}
	set, err := c.SetAuthCookie(user, false, false, 0, "")
	if err != nil {
		t.Fatalf("cookie: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/wp-admin/import.php", nil)
	req.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	rec := httptest.NewRecorder()
	rc := c.ForRequest(rec, req)
	if !(&admin.Handler{}).Handle(rec, req, rc, "/wp-admin/import.php") {
		t.Fatal("import.php not handled")
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Import") || !strings.Contains(body, "importers") {
		t.Fatalf("unexpected body %q", body)
	}
	if !strings.Contains(body, "WordPress") {
		t.Fatalf("expected WordPress importer row: %q", body)
	}
	if !strings.Contains(body, "import-php") {
		t.Fatalf("expected import-php body class: %q", body)
	}
	if !strings.Contains(body, "Run Importer") {
		t.Fatalf("expected Run Importer action: %q", body)
	}
}

func TestImportCapDenied(t *testing.T) {
	c := bootOrSkip(t)
	user, err := c.GetUserBy("login", "author")
	if err != nil || user == nil {
		// Fixture may lack author; synthesize via subscriber-like denial with no caps.
		user, err = c.GetUserBy("login", "admin")
		if err != nil || user == nil {
			t.Fatalf("user: %v", err)
		}
		// Drop to a role without import if possible; otherwise skip when only admin exists.
		if c.UserCan(user, "import") {
			// Use a fresh context with CurrentUser cleared after cookie of a non-import user.
			// Prefer subscriber login when present.
			if sub, err := c.GetUserBy("login", "subscriber"); err == nil && sub != nil {
				user = sub
			} else {
				t.Skip("no non-import user in fixture")
			}
		}
	}
	if c.UserCan(user, "import") {
		t.Skip("fixture user has import cap")
	}
	set, err := c.SetAuthCookie(user, false, false, 0, "")
	if err != nil {
		t.Fatalf("cookie: %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/wp-admin/import.php", nil)
	req.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	rec := httptest.NewRecorder()
	rc := c.ForRequest(rec, req)
	if !(&admin.Handler{}).Handle(rec, req, rc, "/wp-admin/import.php") {
		t.Fatal("import.php not handled")
	}
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status=%d want 403 body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "not allowed to import") {
		t.Fatalf("unexpected denial body: %q", rec.Body.String())
	}
}

func TestImportInvalidNotice(t *testing.T) {
	c := bootOrSkip(t)
	user, err := c.GetUserBy("login", "admin")
	if err != nil || user == nil {
		t.Fatalf("admin: %v", err)
	}
	set, err := c.SetAuthCookie(user, false, false, 0, "")
	if err != nil {
		t.Fatalf("cookie: %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/wp-admin/import.php?invalid=movabletype", nil)
	req.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	rec := httptest.NewRecorder()
	rc := c.ForRequest(rec, req)
	if !(&admin.Handler{}).Handle(rec, req, rc, "/wp-admin/import.php") {
		t.Fatal("import.php not handled")
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "movabletype") || !strings.Contains(body, "invalid or is not installed") {
		t.Fatalf("expected invalid importer notice: %q", body)
	}
}

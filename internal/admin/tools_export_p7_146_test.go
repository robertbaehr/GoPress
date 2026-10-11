package admin_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/admin"
)

// P7.146: export.php populates category/author/date filters + export_add_js.
func TestExportFilterDropdownsP7146(t *testing.T) {
	c := bootOrSkip(t)
	user, err := c.GetUserBy("login", "admin")
	if err != nil || user == nil {
		t.Fatalf("admin: %v", err)
	}
	set, err := c.SetAuthCookie(user, false, false, 0, "")
	if err != nil {
		t.Fatalf("cookie: %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/wp-admin/export.php", nil)
	req.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	rec := httptest.NewRecorder()
	rc := c.ForRequest(rec, req)
	if !(&admin.Handler{}).Handle(rec, req, rc, "/wp-admin/export.php") {
		t.Fatal("export.php not handled")
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d", rec.Code)
	}
	body := rec.Body.String()
	for _, want := range []string{
		`name="cat"`,
		`id="cat"`,
		`name="post_author"`,
		`Uncategorized`,
		`export_add_js`, // may not be in body; check script instead
	} {
		if want == `export_add_js` {
			if !strings.Contains(body, `$('#export-filters')`) && !strings.Contains(body, "export-filters") {
				t.Fatalf("expected export filter JS/markup: %q", body[:min(600, len(body))])
			}
			continue
		}
		if !strings.Contains(body, want) {
			t.Fatalf("missing %q in export body", want)
		}
	}
	if !strings.Contains(body, `slideDown`) {
		t.Fatalf("expected export_add_js slideDown: %q", body[:min(800, len(body))])
	}
}

// P7.146: tools.php shows Export card for export-capable users.
func TestToolsExportCardP7146(t *testing.T) {
	c := bootOrSkip(t)
	user, err := c.GetUserBy("login", "admin")
	if err != nil || user == nil {
		t.Fatalf("admin: %v", err)
	}
	set, err := c.SetAuthCookie(user, false, false, 0, "")
	if err != nil {
		t.Fatalf("cookie: %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/wp-admin/tools.php", nil)
	req.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	rec := httptest.NewRecorder()
	rc := c.ForRequest(rec, req)
	if !(&admin.Handler{}).Handle(rec, req, rc, "/wp-admin/tools.php") {
		t.Fatal("tools.php not handled")
	}
	body := rec.Body.String()
	if !strings.Contains(body, `<h2 class="title">Export</h2>`) {
		t.Fatalf("expected Export card: %q", body[:min(1200, len(body))])
	}
	if !strings.Contains(body, `href="export.php"`) {
		t.Fatalf("expected export.php link in card: %q", body[:min(1200, len(body))])
	}
}

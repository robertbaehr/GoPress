package admin_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/admin"
)

func TestToolsStubScreen(t *testing.T) {
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
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, "<h1>Tools</h1>") {
		t.Fatalf("missing Tools heading: %q", body)
	}
	if !strings.Contains(body, "import.php") || !strings.Contains(body, "export.php") {
		t.Fatalf("expected import/export links: %q", body)
	}
}

func TestToolsPrivacyRedirects(t *testing.T) {
	c := bootOrSkip(t)
	user, err := c.GetUserBy("login", "admin")
	if err != nil || user == nil {
		t.Fatalf("admin: %v", err)
	}
	set, err := c.SetAuthCookie(user, false, false, 0, "")
	if err != nil {
		t.Fatalf("cookie: %v", err)
	}
	cases := []struct {
		page string
		want string
	}{
		{"export_personal_data", "/wp-admin/export-personal-data.php"},
		{"remove_personal_data", "/wp-admin/erase-personal-data.php"},
	}
	for _, tc := range cases {
		req := httptest.NewRequest(http.MethodGet, "/wp-admin/tools.php?page="+tc.page, nil)
		req.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
		rec := httptest.NewRecorder()
		rc := c.ForRequest(rec, req)
		if !(&admin.Handler{}).Handle(rec, req, rc, "/wp-admin/tools.php") {
			t.Fatalf("%s not handled", tc.page)
		}
		if rec.Code != http.StatusMovedPermanently {
			t.Fatalf("%s status=%d", tc.page, rec.Code)
		}
		if loc := rec.Header().Get("Location"); loc != tc.want {
			t.Fatalf("%s Location=%q want %q", tc.page, loc, tc.want)
		}
	}
}

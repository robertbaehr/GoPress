package admin_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/admin"
)

func TestPrivacyStubScreens(t *testing.T) {
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
		path  string
		title string
	}{
		{"/wp-admin/export-personal-data.php", "Export Personal Data"},
		{"/wp-admin/erase-personal-data.php", "Erase Personal Data"},
		{"/wp-admin/options-privacy.php", "Privacy"},
	}
	for _, tc := range cases {
		req := httptest.NewRequest(http.MethodGet, tc.path, nil)
		req.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
		rec := httptest.NewRecorder()
		rc := c.ForRequest(rec, req)
		if !(&admin.Handler{}).Handle(rec, req, rc, tc.path) {
			t.Fatalf("%s not handled", tc.path)
		}
		if rec.Code != http.StatusOK {
			t.Fatalf("%s status=%d body=%s", tc.path, rec.Code, rec.Body.String())
		}
		body := rec.Body.String()
		if !strings.Contains(body, tc.title) {
			t.Fatalf("%s missing title %q in %q", tc.path, tc.title, body)
		}
		if !strings.Contains(body, "privacy-settings") && !strings.Contains(body, "No requests found") {
			t.Fatalf("%s unexpected body %q", tc.path, body)
		}
	}
}

func TestPrivacyExportGeneratesFile(t *testing.T) {
	c := bootOrSkip(t)
	user, err := c.GetUserBy("login", "admin")
	if err != nil || user == nil {
		t.Fatalf("admin: %v", err)
	}
	set, err := c.SetAuthCookie(user, false, false, 0, "")
	if err != nil {
		t.Fatalf("cookie: %v", err)
	}
	form := strings.NewReader("username=privacy-post%40example.com")
	req := httptest.NewRequest(http.MethodPost, "/wp-admin/export-personal-data.php", form)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	rec := httptest.NewRecorder()
	rc := c.ForRequest(rec, req)
	if !(&admin.Handler{}).Handle(rec, req, rc, "/wp-admin/export-personal-data.php") {
		t.Fatal("not handled")
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Export file generated:") {
		t.Fatalf("expected generation note, got %q", body)
	}
	if !strings.Contains(body, "wp-personal-data-exports") {
		t.Fatalf("expected export url path, got %q", body)
	}
}

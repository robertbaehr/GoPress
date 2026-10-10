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

func TestPrivacyEraseAnonymizes(t *testing.T) {
	c := bootOrSkip(t)
	user, err := c.GetUserBy("login", "admin")
	if err != nil || user == nil {
		t.Fatalf("admin: %v", err)
	}
	set, err := c.SetAuthCookie(user, false, false, 0, "")
	if err != nil {
		t.Fatalf("cookie: %v", err)
	}
	form := strings.NewReader("username=erase-post%40example.com")
	req := httptest.NewRequest(http.MethodPost, "/wp-admin/erase-personal-data.php", form)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	rec := httptest.NewRecorder()
	rc := c.ForRequest(rec, req)
	if !(&admin.Handler{}).Handle(rec, req, rc, "/wp-admin/erase-personal-data.php") {
		t.Fatal("not handled")
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Erased personal data for") {
		t.Fatalf("expected erasure note, got %q", body)
	}
	if !strings.Contains(body, "wp-privacy-request-form") {
		t.Fatalf("expected erase form, got %q", body)
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
	if !strings.Contains(body, "Export file generated") {
		t.Fatalf("expected generation note, got %q", body)
	}
	if !strings.Contains(body, "export-download-links") {
		t.Fatalf("expected download links, got %q", body)
	}
	if !strings.Contains(body, "action=download") {
		t.Fatalf("expected download action URL, got %q", body)
	}
	if !strings.Contains(body, "Download JSON") || !strings.Contains(body, "Download HTML") {
		t.Fatalf("expected JSON/HTML download buttons, got %q", body)
	}
}

func TestPrivacyExportDownload(t *testing.T) {
	c := bootOrSkip(t)
	user, err := c.GetUserBy("login", "admin")
	if err != nil || user == nil {
		t.Fatalf("admin: %v", err)
	}
	set, err := c.SetAuthCookie(user, false, false, 0, "")
	if err != nil {
		t.Fatalf("cookie: %v", err)
	}
	form := strings.NewReader("username=privacy-dl%40example.com")
	req := httptest.NewRequest(http.MethodPost, "/wp-admin/export-personal-data.php", form)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	rec := httptest.NewRecorder()
	rc := c.ForRequest(rec, req)
	if !(&admin.Handler{}).Handle(rec, req, rc, "/wp-admin/export-personal-data.php") {
		t.Fatal("not handled")
	}
	body := rec.Body.String()
	idx := strings.Index(body, "action=download&amp;file=")
	if idx < 0 {
		idx = strings.Index(body, "action=download&file=")
	}
	if idx < 0 {
		t.Fatalf("missing download href in %q", body)
	}
	// Prefer JSON download link.
	hrefStart := strings.LastIndex(body[:idx], `href="`)
	if hrefStart < 0 {
		t.Fatal("missing href")
	}
	hrefStart += len(`href="`)
	hrefEnd := strings.Index(body[hrefStart:], `"`)
	href := body[hrefStart : hrefStart+hrefEnd]
	href = strings.ReplaceAll(href, "&amp;", "&")
	req = httptest.NewRequest(http.MethodGet, href, nil)
	req.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	rec = httptest.NewRecorder()
	rc = c.ForRequest(rec, req)
	if !(&admin.Handler{}).Handle(rec, req, rc, "/wp-admin/export-personal-data.php") {
		t.Fatal("download not handled")
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("download status=%d body=%s", rec.Code, rec.Body.String())
	}
	cd := rec.Header().Get("Content-Disposition")
	if !strings.Contains(cd, "attachment") {
		t.Fatalf("expected attachment disposition, got %q", cd)
	}
	if rec.Body.Len() == 0 {
		t.Fatal("empty download body")
	}
}

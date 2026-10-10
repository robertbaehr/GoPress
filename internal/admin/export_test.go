package admin_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/admin"
)

func TestExportStubScreen(t *testing.T) {
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
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Export") || !strings.Contains(body, "export-filters") {
		t.Fatalf("unexpected body %q", body)
	}
	if strings.Contains(body, `disabled="disabled"`) {
		t.Fatalf("download button should be enabled: %q", body)
	}
	if !strings.Contains(body, `name="download" value="true"`) {
		t.Fatalf("expected download=true hidden: %q", body)
	}
}

func TestExportWXRDownload(t *testing.T) {
	c := bootOrSkip(t)
	user, err := c.GetUserBy("login", "admin")
	if err != nil || user == nil {
		t.Fatalf("admin: %v", err)
	}
	set, err := c.SetAuthCookie(user, false, false, 0, "")
	if err != nil {
		t.Fatalf("cookie: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/wp-admin/export.php?download=true&content=all", nil)
	req.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	rec := httptest.NewRecorder()
	rc := c.ForRequest(rec, req)
	if !(&admin.Handler{}).Handle(rec, req, rc, "/wp-admin/export.php") {
		t.Fatal("export.php not handled")
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	ct := rec.Header().Get("Content-Type")
	if !strings.Contains(ct, "xml") {
		t.Fatalf("Content-Type=%q want xml", ct)
	}
	cd := rec.Header().Get("Content-Disposition")
	if !strings.Contains(cd, "attachment") || !strings.Contains(cd, ".xml") {
		t.Fatalf("Content-Disposition=%q", cd)
	}
	body := rec.Body.String()
	for _, want := range []string{
		`<rss version="2.0"`,
		`<wp:wxr_version>1.2</wp:wxr_version>`,
		`<wp:base_blog_url>`,
		`<item>`,
		`<wp:post_type>`,
		`</channel>`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing %q in WXR (len=%d)", want, len(body))
		}
	}
}

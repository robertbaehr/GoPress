package admin_test

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/admin"
)

func TestImportWXRScreenHasUploader(t *testing.T) {
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
	body := rec.Body.String()
	if !strings.Contains(body, "WordPress") || !strings.Contains(body, `enctype="multipart/form-data"`) {
		t.Fatalf("expected WXR uploader: %q", truncate(body, 400))
	}
	if !strings.Contains(body, `name="import_file"`) {
		t.Fatalf("missing import_file input: %q", truncate(body, 400))
	}
}

func TestImportWXRRejectsNonWXR(t *testing.T) {
	c := bootOrSkip(t)
	user, err := c.GetUserBy("login", "admin")
	if err != nil || user == nil {
		t.Fatalf("admin: %v", err)
	}
	set, err := c.SetAuthCookie(user, false, false, 0, "")
	if err != nil {
		t.Fatalf("cookie: %v", err)
	}

	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	part, err := mw.CreateFormFile("import_file", "notes.txt")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = part.Write([]byte("this is not xml"))
	_ = mw.WriteField("import", "wordpress")
	_ = mw.Close()

	req := httptest.NewRequest(http.MethodPost, "/wp-admin/import.php", &body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	rec := httptest.NewRecorder()
	rc := c.ForRequest(rec, req)
	if !(&admin.Handler{}).Handle(rec, req, rc, "/wp-admin/import.php") {
		t.Fatal("import.php not handled")
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d", rec.Code)
	}
	out := rec.Body.String()
	if !strings.Contains(out, "Import rejected") || !strings.Contains(out, "notice-error") {
		t.Fatalf("expected rejection notice: %q", truncate(out, 500))
	}
}

func TestImportWXRParsesMinimal(t *testing.T) {
	c := bootOrSkip(t)
	user, err := c.GetUserBy("login", "admin")
	if err != nil || user == nil {
		t.Fatalf("admin: %v", err)
	}
	set, err := c.SetAuthCookie(user, false, false, 0, "")
	if err != nil {
		t.Fatalf("cookie: %v", err)
	}

	wxr := `<?xml version="1.0" encoding="UTF-8" ?>
<rss version="2.0"
	xmlns:excerpt="http://wordpress.org/export/1.2/excerpt/"
	xmlns:content="http://purl.org/rss/1.0/modules/content/"
	xmlns:dc="http://purl.org/dc/elements/1.1/"
	xmlns:wp="http://wordpress.org/export/1.2/">
<channel>
	<title>Fixture</title>
	<link>http://example.com</link>
	<wp:wxr_version>1.2</wp:wxr_version>
	<item>
		<title>Hello</title>
		<wp:post_type>post</wp:post_type>
	</item>
	<item>
		<title>About</title>
		<wp:post_type>page</wp:post_type>
	</item>
</channel>
</rss>
`
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	part, err := mw.CreateFormFile("import_file", "export.xml")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = part.Write([]byte(wxr))
	_ = mw.WriteField("import", "wordpress")
	_ = mw.Close()

	req := httptest.NewRequest(http.MethodPost, "/wp-admin/import.php", &body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	rec := httptest.NewRecorder()
	rc := c.ForRequest(rec, req)
	if !(&admin.Handler{}).Handle(rec, req, rc, "/wp-admin/import.php") {
		t.Fatal("import.php not handled")
	}
	out := rec.Body.String()
	if !strings.Contains(out, "Parsed WXR") || !strings.Contains(out, "2 item") {
		t.Fatalf("expected parse success: %q", truncate(out, 600))
	}
	if !strings.Contains(out, "notice-success") {
		t.Fatalf("expected success notice: %q", truncate(out, 400))
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

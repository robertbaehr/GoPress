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

// P7.147: import.php lists popular importers fallback; tools.php Import card.
func TestImportPopularImportersFallbackP7147(t *testing.T) {
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
	for _, want := range []string{
		"WordPress",
		"Run Importer",
		"Blogger",
		"LiveJournal",
		"Movable Type and TypePad",
		"RSS",
		"Tumblr",
		"Categories and Tags Converter",
		"importer-install",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing %q in import screen", want)
		}
	}
}

func TestToolsImportCardP7147(t *testing.T) {
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
	if !strings.Contains(body, `<h2 class="title">Import</h2>`) {
		t.Fatalf("expected Import card: %q", body[:min(1200, len(body))])
	}
	if !strings.Contains(body, `href="import.php"`) {
		t.Fatalf("expected import.php link: %q", body[:min(1200, len(body))])
	}
}

func TestImportWXRCountsCategoriesTagsAuthorsP7147(t *testing.T) {
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
	xmlns:wfw="http://wellformedweb.org/CommentAPI/"
	xmlns:dc="http://purl.org/dc/elements/1.1/"
	xmlns:wp="http://wordpress.org/export/1.2/">
<channel>
	<title>Demo Site</title>
	<wp:wxr_version>1.2</wp:wxr_version>
	<wp:author><wp:author_id>1</wp:author_id><wp:author_login>admin</wp:author_login></wp:author>
	<wp:category><wp:term_id>1</wp:term_id><wp:category_nicename>news</wp:category_nicename></wp:category>
	<wp:category><wp:term_id>2</wp:term_id><wp:category_nicename>life</wp:category_nicename></wp:category>
	<wp:tag><wp:term_id>3</wp:term_id><wp:tag_slug>hot</wp:tag_slug></wp:tag>
	<item><title>Hello</title></item>
	<item><title>World</title></item>
</channel>
</rss>`

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	part, err := mw.CreateFormFile("import_file", "demo.xml")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write([]byte(wxr)); err != nil {
		t.Fatal(err)
	}
	_ = mw.WriteField("import", "wordpress")
	_ = mw.Close()

	req := httptest.NewRequest(http.MethodPost, "/wp-admin/import.php", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	rec := httptest.NewRecorder()
	rc := c.ForRequest(rec, req)
	if !(&admin.Handler{}).Handle(rec, req, rc, "/wp-admin/import.php") {
		t.Fatal("import.php not handled")
	}
	body := rec.Body.String()
	for _, want := range []string{
		"2 item(s)",
		"2 categories",
		"1 tag",
		"1 author",
		"Demo Site",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing %q in %q", want, body)
		}
	}
}

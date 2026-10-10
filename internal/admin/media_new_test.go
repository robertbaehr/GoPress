package admin_test

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/admin"
)

func TestMediaNewFormStub(t *testing.T) {
	c := bootOrSkip(t)
	user, err := c.GetUserBy("login", "admin")
	if err != nil || user == nil {
		t.Fatalf("admin: %v", err)
	}
	set, err := c.SetAuthCookie(user, false, false, 0, "")
	if err != nil {
		t.Fatalf("cookie: %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/wp-admin/media-new.php", nil)
	req.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	rec := httptest.NewRecorder()
	rc := c.ForRequest(rec, req)
	if !(&admin.Handler{}).Handle(rec, req, rc, "/wp-admin/media-new.php") {
		t.Fatal("media-new.php not handled")
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, "<h1>Upload New Media</h1>") {
		t.Fatalf("missing heading: %q", body)
	}
	if !strings.Contains(body, `enctype="multipart/form-data"`) || !strings.Contains(body, `name="async-upload"`) {
		t.Fatalf("missing upload form: %q", body)
	}
	if !strings.Contains(body, `name="_wpnonce"`) {
		t.Fatalf("missing nonce: %q", body)
	}
}

func TestMediaNewUploadRedirects(t *testing.T) {
	c := bootOrSkip(t)
	user, err := c.GetUserBy("login", "admin")
	if err != nil || user == nil {
		t.Fatalf("admin: %v", err)
	}
	set, err := c.SetAuthCookie(user, false, false, 0, "")
	if err != nil {
		t.Fatalf("cookie: %v", err)
	}

	// Build an authenticated context to mint the media-form nonce.
	probe := httptest.NewRequest(http.MethodGet, "/wp-admin/media-new.php", nil)
	probe.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	probeRec := httptest.NewRecorder()
	pc := c.ForRequest(probeRec, probe)
	pc.AuthenticateRequest()
	nonce := pc.CreateNonce("media-form")
	if nonce == "" {
		t.Fatal("empty nonce")
	}

	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	_ = mw.WriteField("_wpnonce", nonce)
	_ = mw.WriteField("html-upload", "Upload")
	part, err := mw.CreateFormFile("async-upload", "media-new-test.txt")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write([]byte("media-new-bytes")); err != nil {
		t.Fatal(err)
	}
	_ = mw.Close()

	req := httptest.NewRequest(http.MethodPost, "/wp-admin/media-new.php", &body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	rec := httptest.NewRecorder()
	rc := c.ForRequest(rec, req)
	if !(&admin.Handler{}).Handle(rec, req, rc, "/wp-admin/media-new.php") {
		t.Fatal("not handled")
	}
	if rec.Code != http.StatusFound {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if loc := rec.Header().Get("Location"); loc != "/wp-admin/upload.php" {
		t.Fatalf("Location=%q", loc)
	}
	t.Cleanup(func() {
		_ = os.Remove("/workspace/wp-content/uploads/2024/06/media-new-test.txt")
		if c.DB != nil {
			q, args, _ := c.DB.Prepare("DELETE FROM "+c.DB.Posts()+" WHERE post_title = %s AND post_type = %s", "media-new-test", "attachment")
			_, _ = c.DB.Exec(q, args...)
		}
	})
}

package admin_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/admin"
)

func TestMediaUploadStub(t *testing.T) {
	// P8.31: legacy media-upload.php iframe stub.
	c := bootOrSkip(t)
	user, err := c.GetUserBy("login", "admin")
	if err != nil || user == nil {
		t.Skipf("admin unavailable: %v", err)
	}
	set, err := c.SetAuthCookie(user, false, false, 0, "")
	if err != nil {
		t.Fatalf("SetAuthCookie: %v", err)
	}
	h := &admin.Handler{}
	req := httptest.NewRequest(http.MethodGet, "/wp-admin/media-upload.php?post_id=13&tab=type", nil)
	req.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	rec := httptest.NewRecorder()
	rc := c.ForRequest(rec, req)
	if !h.Handle(rec, req, rc, "/wp-admin/media-upload.php") {
		t.Fatal("media-upload.php not handled")
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	for _, want := range []string{
		`media-upload-php`,
		`name="async-upload"`,
		`name="post_id"`,
		`value="13"`,
		`tab-type`,
		`From Computer`,
		`Media Library`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing %q in %q", want, body)
		}
	}
}

func TestMediaUploadUploadFilesGate(t *testing.T) {
	c := bootOrSkip(t)
	subscriber, err := c.GetUserBy("login", "subscriber")
	if err != nil || subscriber == nil {
		t.Skipf("subscriber unavailable: %v", err)
	}
	set, err := c.SetAuthCookie(subscriber, false, false, 0, "")
	if err != nil {
		t.Fatalf("SetAuthCookie: %v", err)
	}
	h := &admin.Handler{}
	req := httptest.NewRequest(http.MethodGet, "/wp-admin/media-upload.php", nil)
	req.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	rec := httptest.NewRecorder()
	rc := c.ForRequest(rec, req)
	if !h.Handle(rec, req, rc, "/wp-admin/media-upload.php") {
		t.Fatal("media-upload.php not handled")
	}
	if rec.Code != http.StatusForbidden {
		t.Fatalf("subscriber status=%d want 403", rec.Code)
	}
}

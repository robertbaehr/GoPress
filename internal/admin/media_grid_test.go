package admin_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/admin"
)

func TestUploadMediaGridDefault(t *testing.T) {
	c := bootOrSkip(t)
	user, err := c.GetUserBy("login", "admin")
	if err != nil || user == nil {
		t.Fatalf("admin: %v", err)
	}
	set, err := c.SetAuthCookie(user, false, false, 0, "")
	if err != nil {
		t.Fatalf("cookie: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/wp-admin/upload.php", nil)
	req.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	rec := httptest.NewRecorder()
	rc := c.ForRequest(rec, req)
	if !(&admin.Handler{}).Handle(rec, req, rc, "/wp-admin/upload.php") {
		t.Fatal("upload.php not handled")
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d", rec.Code)
	}
	body := rec.Body.String()
	for _, want := range []string{
		`id="wp-media-grid"`,
		"view-switch",
		"view-grid current",
		"attachments",
		"/wp-admin/upload.php?mode=list",
		"/wp-admin/post.php?post=",
		"action=edit",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing %q in %q", want, trunc(body, 700))
		}
	}
	if strings.Contains(body, `id="the-list"`) {
		t.Fatalf("grid mode should not render list table: %q", trunc(body, 400))
	}
}

func TestUploadMediaListMode(t *testing.T) {
	c := bootOrSkip(t)
	user, err := c.GetUserBy("login", "admin")
	if err != nil || user == nil {
		t.Fatalf("admin: %v", err)
	}
	set, err := c.SetAuthCookie(user, false, false, 0, "")
	if err != nil {
		t.Fatalf("cookie: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/wp-admin/upload.php?mode=list", nil)
	req.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	rec := httptest.NewRecorder()
	rc := c.ForRequest(rec, req)
	if !(&admin.Handler{}).Handle(rec, req, rc, "/wp-admin/upload.php") {
		t.Fatal("upload.php not handled")
	}
	body := rec.Body.String()
	if !strings.Contains(body, `id="the-list"`) || !strings.Contains(body, "wp-list-table") {
		t.Fatalf("expected list table: %q", trunc(body, 500))
	}
	if !strings.Contains(body, "view-list current") {
		t.Fatalf("expected list view current: %q", trunc(body, 400))
	}
	if strings.Contains(body, `id="wp-media-grid"`) {
		t.Fatalf("list mode should not render grid: %q", trunc(body, 400))
	}
}

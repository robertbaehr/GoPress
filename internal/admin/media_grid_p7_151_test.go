package admin_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/admin"
	"github.com/robertbaehr/GoPress/internal/wp"
)

// P7.151: grid emits attachments-browser, data-filter, data-type/mime.
func TestUploadMediaGridBrowserChromeP7151(t *testing.T) {
	c := bootOrSkip(t)
	user, err := c.GetUserBy("login", "admin")
	if err != nil || user == nil {
		t.Fatalf("admin: %v", err)
	}
	set, err := c.SetAuthCookie(user, false, false, 0, "")
	if err != nil {
		t.Fatalf("cookie: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/wp-admin/upload.php?mode=grid&attachment-filter=image&s=photo", nil)
	req.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	rec := httptest.NewRecorder()
	rc := c.ForRequest(rec, req)
	if !(&admin.Handler{}).Handle(rec, req, rc, "/wp-admin/upload.php") {
		t.Fatal("upload.php not handled")
	}
	body := rec.Body.String()
	for _, want := range []string{
		`id="wp-media-grid"`,
		`data-search="photo"`,
		`data-filter="image"`,
		`attachments-browser`,
		`mode-grid`,
		`selected="selected">Images`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing %q in %q", want, trunc(body, 900))
		}
	}
}

// P7.151: explicit ?mode= persists media_library_mode usermeta.
func TestUploadMediaModePersistsP7151(t *testing.T) {
	c := bootOrSkip(t)
	user, err := c.GetUserBy("login", "admin")
	if err != nil || user == nil {
		t.Fatalf("admin: %v", err)
	}
	set, err := c.SetAuthCookie(user, false, false, 0, "")
	if err != nil {
		t.Fatalf("cookie: %v", err)
	}
	t.Cleanup(func() {
		_, _, _, _ = c.Meta.UpdateMetadata(wp.MetaUser, user.ID, "media_library_mode", "grid", "")
	})

	req := httptest.NewRequest(http.MethodGet, "/wp-admin/upload.php?mode=list", nil)
	req.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	rec := httptest.NewRecorder()
	rc := c.ForRequest(rec, req)
	if !(&admin.Handler{}).Handle(rec, req, rc, "/wp-admin/upload.php") {
		t.Fatal("upload.php not handled")
	}
	mode, _ := c.Meta.GetMetadata(wp.MetaUser, user.ID, "media_library_mode", true).(string)
	if strings.TrimSpace(mode) != "list" {
		t.Fatalf("media_library_mode=%q want list", mode)
	}

	// Default (no mode) should honor persisted list preference.
	req2 := httptest.NewRequest(http.MethodGet, "/wp-admin/upload.php", nil)
	req2.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	rec2 := httptest.NewRecorder()
	rc2 := c.ForRequest(rec2, req2)
	if !(&admin.Handler{}).Handle(rec2, req2, rc2, "/wp-admin/upload.php") {
		t.Fatal("upload.php not handled")
	}
	body := rec2.Body.String()
	if !strings.Contains(body, `id="the-list"`) {
		t.Fatalf("expected persisted list mode: %q", trunc(body, 500))
	}
	if strings.Contains(body, `id="wp-media-grid"`) {
		t.Fatalf("list preference should not render grid: %q", trunc(body, 400))
	}
}

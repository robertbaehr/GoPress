package admin_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/admin"
)

func TestOptionsMediaSaveViaOptionsPHP(t *testing.T) {
	c := bootOrSkip(t)

	user, err := c.GetUserBy("login", "admin")
	if err != nil || user == nil {
		t.Skipf("admin unavailable: %v", err)
	}
	set, err := c.SetAuthCookie(user, false, false, 0, "")
	if err != nil {
		t.Fatalf("SetAuthCookie: %v", err)
	}

	keys := []string{
		"thumbnail_size_w", "thumbnail_size_h", "thumbnail_crop",
		"medium_size_w", "medium_size_h",
		"large_size_w", "large_size_h",
		"embed_autourls",
	}
	orig := map[string]any{}
	for _, k := range keys {
		orig[k] = c.Options.GetOption(k, "")
	}
	defer func() {
		for _, k := range keys {
			_, _ = c.Options.UpdateOption(k, orig[k], "yes")
		}
	}()

	form := url.Values{}
	form.Set("option_page", "media")
	form.Set("thumbnail_size_w", "160")
	form.Set("thumbnail_size_h", "160")
	form.Set("thumbnail_crop", "1")
	form.Set("medium_size_w", "320")
	form.Set("medium_size_h", "320")
	form.Set("large_size_w", "1200")
	form.Set("large_size_h", "1200")
	// embed_autourls omitted → 0
	form.Set("submit", "Save Changes")

	h := &admin.Handler{}
	req := httptest.NewRequest(http.MethodPost, "/wp-admin/options.php", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	rec := httptest.NewRecorder()
	rc := c.ForRequest(rec, req)
	if !h.Handle(rec, req, rc, "/wp-admin/options.php") {
		t.Fatal("options.php not handled")
	}
	if rec.Code != http.StatusFound {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	loc := rec.Header().Get("Location")
	if !strings.Contains(loc, "options-media.php") || !strings.Contains(loc, "settings-updated=true") {
		t.Fatalf("Location=%q", loc)
	}
	want := map[string]string{
		"thumbnail_size_w": "160",
		"thumbnail_size_h": "160",
		"thumbnail_crop":   "1",
		"medium_size_w":    "320",
		"medium_size_h":    "320",
		"large_size_w":     "1200",
		"large_size_h":     "1200",
		"embed_autourls":   "0",
	}
	for k, v := range want {
		if got := strings.TrimSpace(fmt.Sprint(c.Options.GetOption(k, ""))); got != v {
			t.Fatalf("%s=%v want %s", k, got, v)
		}
	}

	req = httptest.NewRequest(http.MethodGet, "/wp-admin/options-media.php?settings-updated=true", nil)
	req.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	rec = httptest.NewRecorder()
	rc = c.ForRequest(rec, req)
	if !h.Handle(rec, req, rc, "/wp-admin/options-media.php") {
		t.Fatal("options-media.php not handled")
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Settings saved.") {
		t.Fatalf("expected saved notice: %q", body)
	}
	if !strings.Contains(body, `name="option_page"`) || !strings.Contains(body, `value="media"`) {
		t.Fatalf("expected option_page=media: %q", body)
	}
}

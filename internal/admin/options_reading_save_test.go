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

func TestOptionsReadingSaveViaOptionsPHP(t *testing.T) {
	c := bootOrSkip(t)

	user, err := c.GetUserBy("login", "admin")
	if err != nil || user == nil {
		t.Skipf("admin unavailable: %v", err)
	}
	set, err := c.SetAuthCookie(user, false, false, 0, "")
	if err != nil {
		t.Fatalf("SetAuthCookie: %v", err)
	}

	origShow := c.Options.GetOption("show_on_front", "posts")
	origPPP := c.Options.GetOption("posts_per_page", "10")
	origPublic := c.Options.GetOption("blog_public", "1")
	defer func() {
		_, _ = c.Options.UpdateOption("show_on_front", origShow, "yes")
		_, _ = c.Options.UpdateOption("posts_per_page", origPPP, "yes")
		_, _ = c.Options.UpdateOption("blog_public", origPublic, "yes")
	}()

	form := url.Values{}
	form.Set("option_page", "reading")
	form.Set("show_on_front", "posts")
	form.Set("posts_per_page", "7")
	form.Set("blog_public", "0")
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
	if !strings.Contains(loc, "options-reading.php") || !strings.Contains(loc, "settings-updated=true") {
		t.Fatalf("Location=%q", loc)
	}
	if got := strings.TrimSpace(fmt.Sprint(c.Options.GetOption("posts_per_page", ""))); got != "7" {
		t.Fatalf("posts_per_page=%v", got)
	}
	if got := strings.TrimSpace(fmt.Sprint(c.Options.GetOption("blog_public", ""))); got != "0" {
		t.Fatalf("blog_public=%v", got)
	}

	req = httptest.NewRequest(http.MethodGet, "/wp-admin/options-reading.php?settings-updated=true", nil)
	req.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	rec = httptest.NewRecorder()
	rc = c.ForRequest(rec, req)
	if !h.Handle(rec, req, rc, "/wp-admin/options-reading.php") {
		t.Fatal("options-reading.php not handled")
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Settings saved.") {
		t.Fatalf("expected saved notice: %q", body)
	}
	if !strings.Contains(body, `name="option_page"`) || !strings.Contains(body, `value="reading"`) {
		t.Fatalf("expected option_page=reading: %q", body)
	}
}

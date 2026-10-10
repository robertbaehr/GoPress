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

func TestOptionsPermalinkSave(t *testing.T) {
	c := bootOrSkip(t)

	user, err := c.GetUserBy("login", "admin")
	if err != nil || user == nil {
		t.Skipf("admin unavailable: %v", err)
	}
	set, err := c.SetAuthCookie(user, false, false, 0, "")
	if err != nil {
		t.Fatalf("SetAuthCookie: %v", err)
	}

	origStruct := c.Options.GetOption("permalink_structure", "")
	origCat := c.Options.GetOption("category_base", "")
	origTag := c.Options.GetOption("tag_base", "")
	defer func() {
		_, _ = c.Options.UpdateOption("permalink_structure", origStruct, "yes")
		_, _ = c.Options.UpdateOption("category_base", origCat, "yes")
		_, _ = c.Options.UpdateOption("tag_base", origTag, "yes")
	}()

	form := url.Values{}
	form.Set("selection", "/%postname%/")
	form.Set("permalink_structure", "")
	form.Set("category_base", "topics")
	form.Set("tag_base", "labels")
	form.Set("submit", "Save Changes")

	h := &admin.Handler{}
	req := httptest.NewRequest(http.MethodPost, "/wp-admin/options-permalink.php", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	rec := httptest.NewRecorder()
	rc := c.ForRequest(rec, req)
	if !h.Handle(rec, req, rc, "/wp-admin/options-permalink.php") {
		t.Fatal("options-permalink.php not handled")
	}
	if rec.Code != http.StatusFound {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	loc := rec.Header().Get("Location")
	if !strings.Contains(loc, "options-permalink.php") || !strings.Contains(loc, "settings-updated=true") {
		t.Fatalf("Location=%q", loc)
	}
	if got := strings.TrimSpace(fmt.Sprint(c.Options.GetOption("permalink_structure", ""))); got != "/%postname%/" {
		t.Fatalf("permalink_structure=%q", got)
	}
	if got := strings.TrimSpace(fmt.Sprint(c.Options.GetOption("category_base", ""))); got != "topics" {
		t.Fatalf("category_base=%q", got)
	}
	if got := strings.TrimSpace(fmt.Sprint(c.Options.GetOption("tag_base", ""))); got != "labels" {
		t.Fatalf("tag_base=%q", got)
	}

	req = httptest.NewRequest(http.MethodGet, "/wp-admin/options-permalink.php?settings-updated=true", nil)
	req.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	rec = httptest.NewRecorder()
	rc = c.ForRequest(rec, req)
	if !h.Handle(rec, req, rc, "/wp-admin/options-permalink.php") {
		t.Fatal("options-permalink.php not handled")
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Permalink structure updated.") {
		t.Fatalf("expected saved notice: %q", body)
	}
}

func TestOptionsPermalinkSaveCustom(t *testing.T) {
	c := bootOrSkip(t)

	user, err := c.GetUserBy("login", "admin")
	if err != nil || user == nil {
		t.Skipf("admin unavailable: %v", err)
	}
	set, err := c.SetAuthCookie(user, false, false, 0, "")
	if err != nil {
		t.Fatalf("SetAuthCookie: %v", err)
	}

	origStruct := c.Options.GetOption("permalink_structure", "")
	defer func() {
		_, _ = c.Options.UpdateOption("permalink_structure", origStruct, "yes")
	}()

	form := url.Values{}
	form.Set("selection", "custom")
	form.Set("permalink_structure", "blog/%postname%")
	form.Set("submit", "Save Changes")

	h := &admin.Handler{}
	req := httptest.NewRequest(http.MethodPost, "/wp-admin/options-permalink.php", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	rec := httptest.NewRecorder()
	rc := c.ForRequest(rec, req)
	if !h.Handle(rec, req, rc, "/wp-admin/options-permalink.php") {
		t.Fatal("options-permalink.php not handled")
	}
	if rec.Code != http.StatusFound {
		t.Fatalf("status=%d", rec.Code)
	}
	got := strings.TrimSpace(fmt.Sprint(c.Options.GetOption("permalink_structure", "")))
	if got != "/blog/%postname%" {
		t.Fatalf("permalink_structure=%q want /blog/%%postname%%", got)
	}
}
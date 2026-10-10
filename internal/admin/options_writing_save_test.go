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

func TestOptionsWritingSaveViaOptionsPHP(t *testing.T) {
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
		"default_category", "default_post_format",
		"mailserver_url", "mailserver_port", "mailserver_login", "mailserver_pass",
		"default_email_category", "ping_sites",
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
	form.Set("option_page", "writing")
	form.Set("default_category", "2")
	form.Set("default_post_format", "0")
	form.Set("mailserver_url", "pop.example.test")
	form.Set("mailserver_port", "995")
	form.Set("mailserver_login", "secret@example.test")
	form.Set("mailserver_pass", "s3cret")
	form.Set("default_email_category", "3")
	form.Set("ping_sites", "https://rpc.example.test/\nhttps://ping.example.test/")
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
	if !strings.Contains(loc, "options-writing.php") || !strings.Contains(loc, "settings-updated=true") {
		t.Fatalf("Location=%q", loc)
	}
	want := map[string]string{
		"default_category":       "2",
		"mailserver_url":         "pop.example.test",
		"mailserver_port":        "995",
		"mailserver_login":       "secret@example.test",
		"mailserver_pass":        "s3cret",
		"default_email_category": "3",
		"ping_sites":             "https://rpc.example.test/\nhttps://ping.example.test/",
	}
	for k, v := range want {
		if got := strings.TrimSpace(fmt.Sprint(c.Options.GetOption(k, ""))); got != v {
			t.Fatalf("%s=%q want %q", k, got, v)
		}
	}

	req = httptest.NewRequest(http.MethodGet, "/wp-admin/options-writing.php?settings-updated=true", nil)
	req.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	rec = httptest.NewRecorder()
	rc = c.ForRequest(rec, req)
	if !h.Handle(rec, req, rc, "/wp-admin/options-writing.php") {
		t.Fatal("options-writing.php not handled")
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Settings saved.") {
		t.Fatalf("expected saved notice: %q", body)
	}
	if !strings.Contains(body, `name="option_page"`) || !strings.Contains(body, `value="writing"`) {
		t.Fatalf("expected option_page=writing: %q", body)
	}
	for _, wantHTML := range []string{
		`id="wp-settings-section-post-via-email"`,
		`name="mailserver_url"`,
		`name="ping_sites"`,
		`id="wp-settings-section-update-services"`,
		`pop.example.test`,
	} {
		if !strings.Contains(body, wantHTML) {
			t.Fatalf("missing %q in %q", wantHTML, body)
		}
	}
}

func TestOptionsWritingManageOptionsGate(t *testing.T) {
	// P7.103: manage_options gate.
	c := bootOrSkip(t)
	h := &admin.Handler{}

	editor, err := c.GetUserBy("login", "editor")
	if err != nil || editor == nil {
		t.Skipf("editor unavailable: %v", err)
	}
	edSet, err := c.SetAuthCookie(editor, false, false, 0, "")
	if err != nil {
		t.Fatalf("SetAuthCookie editor: %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/wp-admin/options-writing.php", nil)
	req.AddCookie(&http.Cookie{Name: edSet.LoggedInName, Value: edSet.LoggedInValue})
	rec := httptest.NewRecorder()
	rc := c.ForRequest(rec, req)
	if !h.Handle(rec, req, rc, "/wp-admin/options-writing.php") {
		t.Fatal("options-writing.php not handled")
	}
	if rec.Code != http.StatusForbidden {
		t.Fatalf("editor status=%d want 403", rec.Code)
	}
}

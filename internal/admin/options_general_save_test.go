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

func TestOptionsGeneralSaveViaOptionsPHP(t *testing.T) {
	c := bootOrSkip(t)

	user, err := c.GetUserBy("login", "admin")
	if err != nil || user == nil {
		t.Skipf("admin unavailable: %v", err)
	}
	set, err := c.SetAuthCookie(user, false, false, 0, "")
	if err != nil {
		t.Fatalf("SetAuthCookie: %v", err)
	}

	origName := c.Options.GetOption("blogname", "")
	origDesc := c.Options.GetOption("blogdescription", "")
	origEmail := c.Options.GetOption("admin_email", "")
	origDate := c.Options.GetOption("date_format", "F j, Y")
	origTime := c.Options.GetOption("time_format", "g:i a")
	origWeek := c.Options.GetOption("start_of_week", "1")
	origTZ := c.Options.GetOption("timezone_string", "")
	defer func() {
		_, _ = c.Options.UpdateOption("blogname", origName, "yes")
		_, _ = c.Options.UpdateOption("blogdescription", origDesc, "yes")
		_, _ = c.Options.UpdateOption("admin_email", origEmail, "yes")
		_, _ = c.Options.UpdateOption("date_format", origDate, "yes")
		_, _ = c.Options.UpdateOption("time_format", origTime, "yes")
		_, _ = c.Options.UpdateOption("start_of_week", origWeek, "yes")
		_, _ = c.Options.UpdateOption("timezone_string", origTZ, "yes")
	}()

	form := url.Values{}
	form.Set("option_page", "general")
	form.Set("blogname", "GoPress General")
	form.Set("blogdescription", "Just another GoPress site")
	form.Set("admin_email", "admin@example.test")
	form.Set("timezone_string", "UTC")
	form.Set("date_format", "Y-m-d")
	form.Set("time_format", "H:i")
	form.Set("start_of_week", "0")
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
	if !strings.Contains(loc, "options-general.php") || !strings.Contains(loc, "settings-updated=true") {
		t.Fatalf("Location=%q", loc)
	}
	if got := strings.TrimSpace(fmt.Sprint(c.Options.GetOption("blogname", ""))); got != "GoPress General" {
		t.Fatalf("blogname=%v", got)
	}
	if got := strings.TrimSpace(fmt.Sprint(c.Options.GetOption("blogdescription", ""))); got != "Just another GoPress site" {
		t.Fatalf("blogdescription=%v", got)
	}
	if got := strings.TrimSpace(fmt.Sprint(c.Options.GetOption("admin_email", ""))); got != "admin@example.test" {
		t.Fatalf("admin_email=%v", got)
	}
	if got := strings.TrimSpace(fmt.Sprint(c.Options.GetOption("date_format", ""))); got != "Y-m-d" {
		t.Fatalf("date_format=%v", got)
	}
	if got := strings.TrimSpace(fmt.Sprint(c.Options.GetOption("start_of_week", ""))); got != "0" {
		t.Fatalf("start_of_week=%v", got)
	}

	req = httptest.NewRequest(http.MethodGet, "/wp-admin/options-general.php?settings-updated=true", nil)
	req.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	rec = httptest.NewRecorder()
	rc = c.ForRequest(rec, req)
	if !h.Handle(rec, req, rc, "/wp-admin/options-general.php") {
		t.Fatal("options-general.php not handled")
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Settings saved.") {
		t.Fatalf("expected saved notice: %q", body)
	}
	for _, wantHTML := range []string{
		`name="option_page"`,
		`value="general"`,
		`name="blogname"`,
		`name="blogdescription"`,
		`name="admin_email"`,
		`name="timezone_string"`,
		`name="date_format"`,
		`name="time_format"`,
		`name="start_of_week"`,
		`GoPress General`,
		`Just another GoPress site`,
	} {
		if !strings.Contains(body, wantHTML) {
			t.Fatalf("missing %q in %q", wantHTML, body)
		}
	}
}

func TestOptionsGeneralManageOptionsGate(t *testing.T) {
	// P7.105: manage_options gate.
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
	req := httptest.NewRequest(http.MethodGet, "/wp-admin/options-general.php", nil)
	req.AddCookie(&http.Cookie{Name: edSet.LoggedInName, Value: edSet.LoggedInValue})
	rec := httptest.NewRecorder()
	rc := c.ForRequest(rec, req)
	if !h.Handle(rec, req, rc, "/wp-admin/options-general.php") {
		t.Fatal("options-general.php not handled")
	}
	if rec.Code != http.StatusForbidden {
		t.Fatalf("editor status=%d want 403", rec.Code)
	}
}

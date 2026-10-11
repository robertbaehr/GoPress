package admin_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/admin"
)

// P7.160: freedoms.php pagenow chrome + admin_body_class + tab data attr.
func TestFreedomsChromeP7160(t *testing.T) {
	c := bootOrSkip(t)
	t.Cleanup(func() { _ = c.DB.Close() })
	user, err := c.GetUserBy("login", "admin")
	if err != nil || user == nil {
		t.Fatalf("admin: %v", err)
	}
	set, err := c.SetAuthCookie(user, false, false, 0, "")
	if err != nil {
		t.Fatalf("cookie: %v", err)
	}
	ck := &http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue}

	c.Hooks.AddFilter("admin_body_class", func(value any, _ ...any) any {
		return "folded"
	}, 10, 1)

	req := httptest.NewRequest(http.MethodGet, "/wp-admin/freedoms.php?tab=license", nil)
	req.AddCookie(ck)
	rec := httptest.NewRecorder()
	rc := c.ForRequest(rec, req)
	if !(&admin.Handler{}).Handle(rec, req, rc, "/wp-admin/freedoms.php") {
		t.Fatal("freedoms.php not handled")
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d", rec.Code)
	}
	body := rec.Body.String()
	for _, want := range []string{
		`pagenow = "freedoms"`,
		`adminpage = "freedoms-php"`,
		`locale-en-us`,
		` folded`,
		`data-tab="license"`,
		`name="viewport"`,
		`The Four Freedoms`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing %q in %q", want, body[:min(800, len(body))])
		}
	}
	if got := rec.Header().Get("X-GoPress-Freedoms"); got != "stub" {
		t.Fatalf("header=%q", got)
	}
}

// P7.160: privacy-notice redirect still works after chrome deepen.
func TestFreedomsPrivacyNoticeRedirectP7160(t *testing.T) {
	c := bootOrSkip(t)
	t.Cleanup(func() { _ = c.DB.Close() })
	user, err := c.GetUserBy("login", "admin")
	if err != nil || user == nil {
		t.Fatalf("admin: %v", err)
	}
	set, err := c.SetAuthCookie(user, false, false, 0, "")
	if err != nil {
		t.Fatalf("cookie: %v", err)
	}
	ck := &http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue}

	req := httptest.NewRequest(http.MethodGet, "/wp-admin/freedoms.php?privacy-notice", nil)
	req.AddCookie(ck)
	rec := httptest.NewRecorder()
	rc := c.ForRequest(rec, req)
	if !(&admin.Handler{}).Handle(rec, req, rc, "/wp-admin/freedoms.php") {
		t.Fatal("freedoms.php not handled")
	}
	if rec.Code != http.StatusMovedPermanently {
		t.Fatalf("status=%d want 301", rec.Code)
	}
	if loc := rec.Header().Get("Location"); loc != "/wp-admin/options-privacy.php" {
		t.Fatalf("Location=%q", loc)
	}
}

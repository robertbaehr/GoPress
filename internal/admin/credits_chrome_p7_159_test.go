package admin_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/admin"
)

// P7.159: credits.php pagenow chrome + admin_body_class + tab data attr.
func TestCreditsChromeP7159(t *testing.T) {
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

	req := httptest.NewRequest(http.MethodGet, "/wp-admin/credits.php?tab=contributors", nil)
	req.AddCookie(ck)
	rec := httptest.NewRecorder()
	rc := c.ForRequest(rec, req)
	if !(&admin.Handler{}).Handle(rec, req, rc, "/wp-admin/credits.php") {
		t.Fatal("credits.php not handled")
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d", rec.Code)
	}
	body := rec.Body.String()
	for _, want := range []string{
		`pagenow = "credits"`,
		`adminpage = "credits-php"`,
		`locale-en-us`,
		` folded`,
		`data-tab="contributors"`,
		`name="viewport"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing %q in %q", want, body[:min(800, len(body))])
		}
	}
	if got := rec.Header().Get("X-GoPress-Credits"); got != "stub" {
		t.Fatalf("header=%q", got)
	}
}

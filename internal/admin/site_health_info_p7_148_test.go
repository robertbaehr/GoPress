package admin_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/admin"
)

// P7.148: site-health-info accordion sections + clipboard debug text.
func TestSiteHealthInfoAccordionP7148(t *testing.T) {
	c := bootOrSkip(t)
	user, err := c.GetUserBy("login", "admin")
	if err != nil || user == nil {
		t.Fatalf("admin: %v", err)
	}
	set, err := c.SetAuthCookie(user, false, false, 0, "")
	if err != nil {
		t.Fatalf("cookie: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/wp-admin/site-health-info.php", nil)
	req.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	rec := httptest.NewRecorder()
	rc := c.ForRequest(rec, req)
	if !(&admin.Handler{}).Handle(rec, req, rc, "/wp-admin/site-health-info.php") {
		t.Fatal("site-health-info.php not handled")
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	for _, want := range []string{
		`health-check-accordion-trigger`,
		`health-check-section-wp-core`,
		`health-check-section-wp-server`,
		`health-check-section-wp-database`,
		`health-check-section-wp-active-theme`,
		`health-check-section-wp-constants`,
		`hidden="hidden"`,
		`scope="row"`,
		`Environment type`,
		`GoPress (Go)`,
		`data-clipboard-text=`,
		`### WordPress ###`,
		`Active Theme`,
		`P7.148`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing %q in body (len=%d)", want, len(body))
		}
	}
}

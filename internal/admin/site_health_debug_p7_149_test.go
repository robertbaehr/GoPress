package admin_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/admin"
)

// P7.149: site-health-info adds media / active plugins / filesystem debug sections.
func TestSiteHealthInfoDebugSectionsP7149(t *testing.T) {
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
		t.Fatalf("status=%d", rec.Code)
	}
	body := rec.Body.String()
	for _, want := range []string{
		`health-check-section-wp-media`,
		`health-check-section-wp-plugins-active`,
		`health-check-section-wp-filesystem`,
		`Media Handling`,
		`Active Plugins`,
		`Filesystem Permissions`,
		`Active editor`,
		`Organize uploads by year/month`,
		`The uploads directory`,
		`### Media Handling ###`,
		`### Filesystem Permissions ###`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing %q in body (len=%d)", want, len(body))
		}
	}
}

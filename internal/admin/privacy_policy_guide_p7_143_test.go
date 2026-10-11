package admin_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/admin"
)

// P7.143: Introduction accordion starts collapsed; privacy-tools enqueue marker.
func TestPrivacyPolicyGuideCollapsedAccordionP7143(t *testing.T) {
	c := bootOrSkip(t)
	user, err := c.GetUserBy("login", "admin")
	if err != nil || user == nil {
		t.Fatalf("admin: %v", err)
	}
	set, err := c.SetAuthCookie(user, false, false, 0, "")
	if err != nil {
		t.Fatalf("cookie: %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/wp-admin/privacy-policy-guide.php", nil)
	req.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	rec := httptest.NewRecorder()
	rc := c.ForRequest(rec, req)
	if !(&admin.Handler{}).Handle(rec, req, rc, "/wp-admin/privacy-policy-guide.php") {
		t.Fatal("privacy-policy-guide.php not handled")
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d", rec.Code)
	}
	body := rec.Body.String()
	for _, want := range []string{
		`id="privacy-tools-js-extra"`,
		`aria-expanded="false"`,
		`id="privacy-settings-accordion-block-privacy-policy-guide"`,
		`hidden="hidden"`,
		"P7.143",
		"privacy-settings privacy-policy-guide-php",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing %q", want)
		}
	}
	// Introduction trigger must be collapsed; do not keep legacy expanded=true default.
	if strings.Contains(body, `aria-controls="privacy-settings-accordion-block-privacy-policy-guide" type="button">`) &&
		strings.Contains(body, `aria-expanded="true" class="privacy-settings-accordion-trigger" aria-controls="privacy-settings-accordion-block-privacy-policy-guide"`) {
		t.Fatalf("introduction accordion should start collapsed")
	}
	if strings.Contains(body, `aria-expanded="true" class="privacy-settings-accordion-trigger" aria-controls="privacy-settings-accordion-block-privacy-policy-guide"`) {
		t.Fatalf("introduction accordion should start collapsed (aria-expanded=false)")
	}
}

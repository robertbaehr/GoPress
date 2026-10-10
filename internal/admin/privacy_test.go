package admin_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/admin"
)

func TestPrivacyStubScreens(t *testing.T) {
	c := bootOrSkip(t)
	user, err := c.GetUserBy("login", "admin")
	if err != nil || user == nil {
		t.Fatalf("admin: %v", err)
	}
	set, err := c.SetAuthCookie(user, false, false, 0, "")
	if err != nil {
		t.Fatalf("cookie: %v", err)
	}

	cases := []struct {
		path  string
		title string
	}{
		{"/wp-admin/export-personal-data.php", "Export Personal Data"},
		{"/wp-admin/erase-personal-data.php", "Erase Personal Data"},
		{"/wp-admin/options-privacy.php", "Privacy"},
		{"/wp-admin/privacy-policy-guide.php", "Privacy Policy Guide"},
	}
	for _, tc := range cases {
		req := httptest.NewRequest(http.MethodGet, tc.path, nil)
		req.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
		rec := httptest.NewRecorder()
		rc := c.ForRequest(rec, req)
		if !(&admin.Handler{}).Handle(rec, req, rc, tc.path) {
			t.Fatalf("%s not handled", tc.path)
		}
		if rec.Code != http.StatusOK {
			t.Fatalf("%s status=%d body=%s", tc.path, rec.Code, rec.Body.String())
		}
		body := rec.Body.String()
		if !strings.Contains(body, tc.title) {
			t.Fatalf("%s missing title %q in %q", tc.path, tc.title, body)
		}
		if !strings.Contains(body, "privacy-settings") && !strings.Contains(body, "No requests found") && !strings.Contains(body, "Policy Guide") {
			t.Fatalf("%s unexpected body %q", tc.path, body)
		}
	}
}

func TestOptionsPrivacyLinksPolicyPage(t *testing.T) {
	c := bootOrSkip(t)
	user, err := c.GetUserBy("login", "admin")
	if err != nil || user == nil {
		t.Fatalf("admin: %v", err)
	}
	set, err := c.SetAuthCookie(user, false, false, 0, "")
	if err != nil {
		t.Fatalf("cookie: %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/wp-admin/options-privacy.php", nil)
	req.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	rec := httptest.NewRecorder()
	rc := c.ForRequest(rec, req)
	if !(&admin.Handler{}).Handle(rec, req, rc, "/wp-admin/options-privacy.php") {
		t.Fatal("options-privacy.php not handled")
	}
	body := rec.Body.String()
	if !strings.Contains(body, "privacy-policy-page-link") {
		t.Fatalf("expected privacy policy page link, got %q", body)
	}
	if !strings.Contains(body, "post.php?post=3") {
		t.Fatalf("expected edit link for policy page, got %q", body)
	}
	if !strings.Contains(body, `name="page_for_privacy_policy"`) || !strings.Contains(body, `value="set-privacy-page"`) {
		t.Fatalf("expected privacy page save form, got %q", body)
	}
}

func TestOptionsPrivacySavePolicyPage(t *testing.T) {
	c := bootOrSkip(t)
	user, err := c.GetUserBy("login", "admin")
	if err != nil || user == nil {
		t.Fatalf("admin: %v", err)
	}
	set, err := c.SetAuthCookie(user, false, false, 0, "")
	if err != nil {
		t.Fatalf("cookie: %v", err)
	}
	h := &admin.Handler{}
	// Switch to Sample Page (ID 2).
	form := strings.NewReader("action=set-privacy-page&page_for_privacy_policy=2")
	req := httptest.NewRequest(http.MethodPost, "/wp-admin/options-privacy.php", form)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	rec := httptest.NewRecorder()
	rc := c.ForRequest(rec, req)
	if !h.Handle(rec, req, rc, "/wp-admin/options-privacy.php") {
		t.Fatal("not handled")
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Privacy Policy page updated successfully") {
		t.Fatalf("expected success notice: %q", body)
	}
	if !strings.Contains(body, `value="2" selected`) && !strings.Contains(body, "selected=\"selected\">Sample Page") {
		t.Fatalf("expected page 2 selected: %q", body)
	}
	// Restore Privacy Policy page (ID 3).
	form = strings.NewReader("action=set-privacy-page&page_for_privacy_policy=3")
	req = httptest.NewRequest(http.MethodPost, "/wp-admin/options-privacy.php", form)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	rec = httptest.NewRecorder()
	rc = c.ForRequest(rec, req)
	if !h.Handle(rec, req, rc, "/wp-admin/options-privacy.php") {
		t.Fatal("restore not handled")
	}
}

func TestPrivacyPolicyGuideSections(t *testing.T) {
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
		"Privacy Policy Guide",
		"Introduction",
		"Who we are",
		"Comments",
		"Cookies",
		"privacy-settings-tab",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing %q", want)
		}
	}
}

func TestPrivacyEraseAnonymizes(t *testing.T) {
	c := bootOrSkip(t)
	user, err := c.GetUserBy("login", "admin")
	if err != nil || user == nil {
		t.Fatalf("admin: %v", err)
	}
	set, err := c.SetAuthCookie(user, false, false, 0, "")
	if err != nil {
		t.Fatalf("cookie: %v", err)
	}
	// Unchecked confirmation → erase immediately (admin-confirmed).
	form := strings.NewReader("username=erase-post%40example.com")
	req := httptest.NewRequest(http.MethodPost, "/wp-admin/erase-personal-data.php", form)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	rec := httptest.NewRecorder()
	rc := c.ForRequest(rec, req)
	if !(&admin.Handler{}).Handle(rec, req, rc, "/wp-admin/erase-personal-data.php") {
		t.Fatal("not handled")
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Erased personal data for") {
		t.Fatalf("expected erasure note, got %q", body)
	}
	if !strings.Contains(body, "wp-privacy-request-form") {
		t.Fatalf("expected erase form, got %q", body)
	}
	if !strings.Contains(body, "send_confirmation_email") {
		t.Fatalf("expected confirmation checkbox, got %q", body)
	}
}

func TestPrivacyEraseConfirmationFlow(t *testing.T) {
	c := bootOrSkip(t)
	user, err := c.GetUserBy("login", "admin")
	if err != nil || user == nil {
		t.Fatalf("admin: %v", err)
	}
	set, err := c.SetAuthCookie(user, false, false, 0, "")
	if err != nil {
		t.Fatalf("cookie: %v", err)
	}
	h := &admin.Handler{}
	form := strings.NewReader("username=erase-confirm%40example.com&send_confirmation_email=1")
	req := httptest.NewRequest(http.MethodPost, "/wp-admin/erase-personal-data.php", form)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	rec := httptest.NewRecorder()
	rc := c.ForRequest(rec, req)
	if !h.Handle(rec, req, rc, "/wp-admin/erase-personal-data.php") {
		t.Fatal("not handled")
	}
	body := rec.Body.String()
	if strings.Contains(body, "Erased personal data for") {
		t.Fatalf("should not erase yet: %q", body)
	}
	if !strings.Contains(body, "Confirmation required") || !strings.Contains(body, "Pending confirmation") {
		t.Fatalf("expected pending confirm UI: %q", body)
	}
	if !strings.Contains(body, `name="action" value="confirm_erasure"`) {
		t.Fatalf("expected confirm form: %q", body)
	}

	confirm := strings.NewReader("action=confirm_erasure&email=erase-confirm%40example.com")
	req = httptest.NewRequest(http.MethodPost, "/wp-admin/erase-personal-data.php", confirm)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	rec = httptest.NewRecorder()
	rc = c.ForRequest(rec, req)
	if !h.Handle(rec, req, rc, "/wp-admin/erase-personal-data.php") {
		t.Fatal("confirm not handled")
	}
	body = rec.Body.String()
	if !strings.Contains(body, "Erased personal data for erase-confirm@example.com") {
		t.Fatalf("expected erasure after confirm: %q", body)
	}
}

func TestPrivacyExportUIPolish(t *testing.T) {
	c := bootOrSkip(t)
	user, err := c.GetUserBy("login", "admin")
	if err != nil || user == nil {
		t.Fatalf("admin: %v", err)
	}
	set, err := c.SetAuthCookie(user, false, false, 0, "")
	if err != nil {
		t.Fatalf("cookie: %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/wp-admin/export-personal-data.php", nil)
	req.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	rec := httptest.NewRecorder()
	rc := c.ForRequest(rec, req)
	if !(&admin.Handler{}).Handle(rec, req, rc, "/wp-admin/export-personal-data.php") {
		t.Fatal("not handled")
	}
	body := rec.Body.String()
	for _, want := range []string{
		"Add Data Export Request",
		"username_or_email_for_privacy_request",
		"send_confirmation_email",
		"Send personal data export confirmation email.",
		"wp-header-end",
		"nosubsub",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing %q in %q", want, body)
		}
	}
}

func TestPrivacyExportGeneratesFile(t *testing.T) {
	c := bootOrSkip(t)
	user, err := c.GetUserBy("login", "admin")
	if err != nil || user == nil {
		t.Fatalf("admin: %v", err)
	}
	set, err := c.SetAuthCookie(user, false, false, 0, "")
	if err != nil {
		t.Fatalf("cookie: %v", err)
	}
	// Unchecked send_confirmation_email → generate immediately (P7.39).
	form := strings.NewReader("username_or_email_for_privacy_request=privacy-post%40example.com")
	req := httptest.NewRequest(http.MethodPost, "/wp-admin/export-personal-data.php", form)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	rec := httptest.NewRecorder()
	rc := c.ForRequest(rec, req)
	if !(&admin.Handler{}).Handle(rec, req, rc, "/wp-admin/export-personal-data.php") {
		t.Fatal("not handled")
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Export file generated") {
		t.Fatalf("expected generation note, got %q", body)
	}
	if !strings.Contains(body, "export-download-links") {
		t.Fatalf("expected download links, got %q", body)
	}
	if !strings.Contains(body, "action=download") {
		t.Fatalf("expected download action URL, got %q", body)
	}
	if !strings.Contains(body, "Download JSON") || !strings.Contains(body, "Download HTML") {
		t.Fatalf("expected JSON/HTML download buttons, got %q", body)
	}
}

func TestPrivacyExportConfirmStep(t *testing.T) {
	c := bootOrSkip(t)
	user, err := c.GetUserBy("login", "admin")
	if err != nil || user == nil {
		t.Fatalf("admin: %v", err)
	}
	set, err := c.SetAuthCookie(user, false, false, 0, "")
	if err != nil {
		t.Fatalf("cookie: %v", err)
	}
	h := &admin.Handler{}
	form := strings.NewReader("username_or_email_for_privacy_request=export-confirm%40example.com&send_confirmation_email=1")
	req := httptest.NewRequest(http.MethodPost, "/wp-admin/export-personal-data.php", form)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	rec := httptest.NewRecorder()
	rc := c.ForRequest(rec, req)
	if !h.Handle(rec, req, rc, "/wp-admin/export-personal-data.php") {
		t.Fatal("not handled")
	}
	body := rec.Body.String()
	if !strings.Contains(body, "export-confirm-pending") || !strings.Contains(body, "Pending confirmation") {
		t.Fatalf("expected pending confirm UI: %q", body)
	}
	confirm := strings.NewReader("action=confirm_export&email=export-confirm%40example.com")
	req = httptest.NewRequest(http.MethodPost, "/wp-admin/export-personal-data.php", confirm)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	rec = httptest.NewRecorder()
	rc = c.ForRequest(rec, req)
	if !h.Handle(rec, req, rc, "/wp-admin/export-personal-data.php") {
		t.Fatal("confirm not handled")
	}
	body = rec.Body.String()
	if !strings.Contains(body, "Export file generated for export-confirm@example.com") {
		t.Fatalf("expected export after confirm: %q", body)
	}
}

func TestPrivacyExportDownload(t *testing.T) {
	c := bootOrSkip(t)
	user, err := c.GetUserBy("login", "admin")
	if err != nil || user == nil {
		t.Fatalf("admin: %v", err)
	}
	set, err := c.SetAuthCookie(user, false, false, 0, "")
	if err != nil {
		t.Fatalf("cookie: %v", err)
	}
	form := strings.NewReader("username_or_email_for_privacy_request=privacy-dl%40example.com")
	req := httptest.NewRequest(http.MethodPost, "/wp-admin/export-personal-data.php", form)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	rec := httptest.NewRecorder()
	rc := c.ForRequest(rec, req)
	if !(&admin.Handler{}).Handle(rec, req, rc, "/wp-admin/export-personal-data.php") {
		t.Fatal("not handled")
	}
	body := rec.Body.String()
	idx := strings.Index(body, "action=download&amp;file=")
	if idx < 0 {
		idx = strings.Index(body, "action=download&file=")
	}
	if idx < 0 {
		t.Fatalf("missing download href in %q", body)
	}
	// Prefer JSON download link.
	hrefStart := strings.LastIndex(body[:idx], `href="`)
	if hrefStart < 0 {
		t.Fatal("missing href")
	}
	hrefStart += len(`href="`)
	hrefEnd := strings.Index(body[hrefStart:], `"`)
	href := body[hrefStart : hrefStart+hrefEnd]
	href = strings.ReplaceAll(href, "&amp;", "&")
	req = httptest.NewRequest(http.MethodGet, href, nil)
	req.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	rec = httptest.NewRecorder()
	rc = c.ForRequest(rec, req)
	if !(&admin.Handler{}).Handle(rec, req, rc, "/wp-admin/export-personal-data.php") {
		t.Fatal("download not handled")
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("download status=%d body=%s", rec.Code, rec.Body.String())
	}
	cd := rec.Header().Get("Content-Disposition")
	if !strings.Contains(cd, "attachment") {
		t.Fatalf("expected attachment disposition, got %q", cd)
	}
	if rec.Body.Len() == 0 {
		t.Fatal("empty download body")
	}
}

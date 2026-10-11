package admin_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/admin"
	"github.com/robertbaehr/GoPress/internal/wp"
)

// P7.150: privacy requests table exposes bulk actions + sortable column headers.
func TestPrivacyRequestsBulkAndSortableUI_P7150(t *testing.T) {
	c := bootOrSkip(t)
	user, err := c.GetUserBy("login", "admin")
	if err != nil || user == nil {
		t.Fatalf("admin: %v", err)
	}
	set, err := c.SetAuthCookie(user, false, false, 0, "")
	if err != nil {
		t.Fatalf("cookie: %v", err)
	}

	email := "privacy-bulk-ui-p7150@example.com"
	reqRow, err := wp.CreateUserRequest(c, email, "remove_personal_data", "pending")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	t.Cleanup(func() { _ = wp.DeleteUserRequest(c, reqRow.ID) })

	req := httptest.NewRequest(http.MethodGet, "/wp-admin/erase-personal-data.php?orderby=requester&order=asc", nil)
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
	for _, want := range []string{
		`class="privacy-requests-bulk-form"`,
		`id="bulk-action-selector-top"`,
		`name="action"`,
		`value="resend"`,
		`value="complete"`,
		`value="delete"`,
		`name="request_id[]"`,
		`name="_wpnonce"`,
		`value="bulk-privacy_requests"`,
		`orderby=requester`,
		`orderby=requested`,
		`sorting-indicator`,
		email,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing %q in erase page", want)
		}
	}
}

// P7.150: bulk complete + delete process_bulk_action.
func TestPrivacyRequestsBulkCompleteDelete_P7150(t *testing.T) {
	c := bootOrSkip(t)
	user, err := c.GetUserBy("login", "admin")
	if err != nil || user == nil {
		t.Fatalf("admin: %v", err)
	}
	set, err := c.SetAuthCookie(user, false, false, 0, "")
	if err != nil {
		t.Fatalf("cookie: %v", err)
	}

	email := "privacy-bulk-del-p7150@example.com"
	reqRow, err := wp.CreateUserRequest(c, email, "export_personal_data", "pending")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	id := reqRow.ID
	t.Cleanup(func() { _ = wp.DeleteUserRequest(c, id) })

	form := url.Values{}
	form.Set("action", "complete")
	form.Set("_wpnonce", "bulk-privacy_requests")
	form.Add("request_id[]", strconv.FormatInt(id, 10))
	req := httptest.NewRequest(http.MethodPost, "/wp-admin/export-personal-data.php", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	rec := httptest.NewRecorder()
	rc := c.ForRequest(rec, req)
	if !(&admin.Handler{}).Handle(rec, req, rc, "/wp-admin/export-personal-data.php") {
		t.Fatal("not handled")
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("complete status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "marked as complete") {
		t.Fatalf("expected complete notice: %s", rec.Body.String())
	}
	got, err := wp.GetUserRequest(c, id)
	if err != nil {
		t.Fatalf("GetUserRequest: %v", err)
	}
	if got.Status != "request-completed" {
		t.Fatalf("status=%q want request-completed", got.Status)
	}

	form = url.Values{}
	form.Set("action", "delete")
	form.Set("_wpnonce", "bulk-privacy_requests")
	form.Add("request_id[]", strconv.FormatInt(id, 10))
	req = httptest.NewRequest(http.MethodPost, "/wp-admin/export-personal-data.php", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	rec = httptest.NewRecorder()
	rc = c.ForRequest(rec, req)
	if !(&admin.Handler{}).Handle(rec, req, rc, "/wp-admin/export-personal-data.php") {
		t.Fatal("not handled")
	}
	if !strings.Contains(rec.Body.String(), "deleted successfully") {
		t.Fatalf("expected delete notice: %s", rec.Body.String())
	}
	if _, err := wp.GetUserRequest(c, id); err == nil {
		t.Fatal("expected deleted request to be gone")
	}
}

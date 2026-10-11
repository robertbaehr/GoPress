package admin_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/admin"
	"github.com/robertbaehr/GoPress/internal/wp"
)

// P7.144: erase-personal-data search box, privacy-tools marker, hr before table.
func TestErasePersonalDataSearchBoxP7144(t *testing.T) {
	c := bootOrSkip(t)
	user, err := c.GetUserBy("login", "admin")
	if err != nil || user == nil {
		t.Fatalf("admin: %v", err)
	}
	set, err := c.SetAuthCookie(user, false, false, 0, "")
	if err != nil {
		t.Fatalf("cookie: %v", err)
	}

	// Seed a pending erase request so search has a hit.
	email := "erase-search-p7144@example.com"
	if _, err := wp.CreateUserRequest(c, email, "remove_personal_data", "pending"); err != nil {
		t.Fatalf("create request: %v", err)
	}
	t.Cleanup(func() {
		if id := wp.FindOpenUserRequestID(c, email, "remove_personal_data"); id > 0 {
			_ = wp.CompleteUserRequest(c, id)
		}
	})

	req := httptest.NewRequest(http.MethodGet, "/wp-admin/erase-personal-data.php?s=erase-search-p7144", nil)
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
		`id="privacy-tools-js-extra"`,
		`class="search-box"`,
		`id="request-search-input"`,
		`name="s"`,
		`value="erase-search-p7144"`,
		`Search Requests`,
		email,
		`<hr />`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing %q in erase page", want)
		}
	}

	// Negative search should yield empty table body.
	req = httptest.NewRequest(http.MethodGet, "/wp-admin/erase-personal-data.php?s=no-such-requester-zzz", nil)
	req.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	rec = httptest.NewRecorder()
	rc = c.ForRequest(rec, req)
	if !(&admin.Handler{}).Handle(rec, req, rc, "/wp-admin/erase-personal-data.php") {
		t.Fatal("not handled")
	}
	body = rec.Body.String()
	if !strings.Contains(body, "No requests found.") {
		t.Fatalf("expected empty search results: %q", body)
	}
	if strings.Contains(body, email) {
		t.Fatalf("search should hide non-matching email")
	}
}

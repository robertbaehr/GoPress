package admin_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/robertbaehr/GoPress/internal/admin"
)

func TestPrivacyRequestsTableColumns(t *testing.T) {
	table := admin.PrivacyRequestsTable{
		RequestType: "export_personal_data",
		Rows: []admin.PrivacyRequest{
			{
				Email:     "user@example.com",
				Status:    "request-pending",
				Requested: time.Date(2026, 10, 10, 12, 0, 0, 0, time.UTC),
				NextSteps: "Waiting for confirmation",
			},
		},
	}
	cols := table.Columns()
	for _, key := range []string{"cb", "email", "status", "created_timestamp", "next_steps"} {
		if cols[key] == "" {
			t.Fatalf("missing column %q", key)
		}
	}
	htmlOut := table.RenderHTML()
	for _, want := range []string{
		"privacy-requests",
		"requests-table",
		"column-email",
		"column-status",
		"column-created_timestamp",
		"column-next_steps",
		"user@example.com",
		"Pending confirmation",
		"request-pending",
		"Waiting for confirmation",
		`name="request_id[]"`,
	} {
		if !strings.Contains(htmlOut, want) {
			t.Fatalf("missing %q in %q", want, trunc(htmlOut, 500))
		}
	}
}

func TestExportPersonalDataUsesPrivacyRequestsTable(t *testing.T) {
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
		"requests-table",
		"privacy-requests",
		"column-email",
		"column-status",
		"column-created_timestamp",
		"column-next_steps",
		`name="request_id[]"`,
	} {
		if !strings.Contains(body, want) {
			// Empty table uses colspan no-items; populated table has checkboxes.
			if want == `name="request_id[]"` && strings.Contains(body, "No requests found") {
				continue
			}
			t.Fatalf("missing %q in %q", want, trunc(body, 600))
		}
	}
}

func TestErasePersonalDataUsesPrivacyRequestsTable(t *testing.T) {
	c := bootOrSkip(t)
	user, err := c.GetUserBy("login", "admin")
	if err != nil || user == nil {
		t.Fatalf("admin: %v", err)
	}
	set, err := c.SetAuthCookie(user, false, false, 0, "")
	if err != nil {
		t.Fatalf("cookie: %v", err)
	}
	form := strings.NewReader("username=erase-table%40example.com&send_confirmation_email=1")
	req := httptest.NewRequest(http.MethodPost, "/wp-admin/erase-personal-data.php", form)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	rec := httptest.NewRecorder()
	rc := c.ForRequest(rec, req)
	if !(&admin.Handler{}).Handle(rec, req, rc, "/wp-admin/erase-personal-data.php") {
		t.Fatal("not handled")
	}
	body := rec.Body.String()
	for _, want := range []string{
		"requests-table",
		"erase-table@example.com",
		"request-pending",
		"Pending confirmation",
		"column-created_timestamp",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing %q in %q", want, trunc(body, 700))
		}
	}
}

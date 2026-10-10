package admin_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/robertbaehr/GoPress/internal/admin"
)

func TestUsersBulkDeleteStub(t *testing.T) {
	c := bootOrSkip(t)
	t.Cleanup(func() { _ = c.DB.Close() })

	adminUser, err := c.GetUserBy("login", "admin")
	if err != nil || adminUser == nil {
		t.Fatalf("admin: %v", err)
	}
	set, err := c.SetAuthCookie(adminUser, false, false, 0, "")
	if err != nil {
		t.Fatalf("cookie: %v", err)
	}
	ck := &http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue}

	login := fmt.Sprintf("bulkdel_%d", time.Now().UnixNano()%1_000_000)
	email := login + "@example.com"
	now := time.Now().UTC().Format("2006-01-02 15:04:05")
	q, args, err := c.DB.Prepare(
		"INSERT INTO "+c.DB.Users()+
			" (user_login, user_pass, user_nicename, user_email, user_url, user_registered, user_activation_key, user_status, display_name) VALUES "+
			"(%s, %s, %s, %s, %s, %s, %s, %d, %s)",
		login, "x", login, email, "", now, "", 0, login,
	)
	if err != nil {
		t.Fatal(err)
	}
	res, err := c.DB.Exec(q, args...)
	if err != nil {
		t.Fatal(err)
	}
	uid, err := res.LastInsertId()
	if err != nil || uid <= 0 {
		t.Fatalf("insert user: %v id=%d", err, uid)
	}
	t.Cleanup(func() {
		dq, dargs, _ := c.DB.Prepare("DELETE FROM "+c.DB.Users()+" WHERE ID = %d", uid)
		_, _ = c.DB.Exec(dq, dargs...)
		dq, dargs, _ = c.DB.Prepare("DELETE FROM "+c.DB.UserMeta()+" WHERE user_id = %d", uid)
		_, _ = c.DB.Exec(dq, dargs...)
	})

	// List includes bulkactions + checkbox.
	req := httptest.NewRequest(http.MethodGet, "/wp-admin/users.php", nil)
	req.AddCookie(ck)
	rec := httptest.NewRecorder()
	rc := c.ForRequest(rec, req)
	if !(&admin.Handler{}).Handle(rec, req, rc, "/wp-admin/users.php") {
		t.Fatal("expected handle")
	}
	body := rec.Body.String()
	for _, want := range []string{
		`name="users[]"`,
		`id="bulk-action-selector-top"`,
		`value="delete"`,
		`id="doaction"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing %q in users list", want)
		}
	}

	// Confirm screen.
	req = httptest.NewRequest(http.MethodGet, fmt.Sprintf("/wp-admin/users.php?action=delete&users[]=%d", uid), nil)
	req.AddCookie(ck)
	rec = httptest.NewRecorder()
	rc = c.ForRequest(rec, req)
	if !(&admin.Handler{}).Handle(rec, req, rc, "/wp-admin/users.php") {
		t.Fatal("expected handle")
	}
	body = rec.Body.String()
	if !strings.Contains(body, "Delete Users") || !strings.Contains(body, `value="dodelete"`) {
		t.Fatalf("confirm page missing markers: %q", body[:min(len(body), 400)])
	}

	// Perform delete.
	form := url.Values{}
	form.Set("action", "dodelete")
	form.Set("delete_option", "delete")
	form.Add("users[]", fmt.Sprintf("%d", uid))
	req = httptest.NewRequest(http.MethodPost, "/wp-admin/users.php", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(ck)
	rec = httptest.NewRecorder()
	rc = c.ForRequest(rec, req)
	if !(&admin.Handler{}).Handle(rec, req, rc, "/wp-admin/users.php") {
		t.Fatal("expected handle")
	}
	if rec.Code != http.StatusFound {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	loc := rec.Header().Get("Location")
	if !strings.Contains(loc, "update=del") || !strings.Contains(loc, "delete_count=1") {
		t.Fatalf("location=%q", loc)
	}

	gone, _ := c.GetUserBy("id", uid)
	if gone != nil {
		t.Fatalf("user still present: %+v", gone)
	}
	checkQ, checkArgs, err := c.DB.Prepare("SELECT ID FROM "+c.DB.Users()+" WHERE ID = %d LIMIT 1", uid)
	if err != nil {
		t.Fatal(err)
	}
	row, err := c.DB.GetRow(checkQ, checkArgs...)
	if err != nil {
		t.Fatal(err)
	}
	if row != nil {
		t.Fatalf("user row still in DB: %+v", row)
	}
}

func TestUsersBulkDeleteSkipsSelf(t *testing.T) {
	c := bootOrSkip(t)
	t.Cleanup(func() { _ = c.DB.Close() })
	adminUser, err := c.GetUserBy("login", "admin")
	if err != nil || adminUser == nil {
		t.Fatalf("admin: %v", err)
	}
	set, err := c.SetAuthCookie(adminUser, false, false, 0, "")
	if err != nil {
		t.Fatalf("cookie: %v", err)
	}
	ck := &http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue}

	form := url.Values{}
	form.Set("action", "dodelete")
	form.Set("delete_option", "delete")
	form.Add("users[]", fmt.Sprintf("%d", adminUser.ID))
	req := httptest.NewRequest(http.MethodPost, "/wp-admin/users.php", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(ck)
	rec := httptest.NewRecorder()
	rc := c.ForRequest(rec, req)
	if !(&admin.Handler{}).Handle(rec, req, rc, "/wp-admin/users.php") {
		t.Fatal("expected handle")
	}
	if rec.Code != http.StatusFound {
		t.Fatalf("status=%d", rec.Code)
	}
	if !strings.Contains(rec.Header().Get("Location"), "err_admin_del") {
		t.Fatalf("location=%q", rec.Header().Get("Location"))
	}
	still, _ := c.GetUserBy("id", adminUser.ID)
	if still == nil {
		t.Fatal("admin was deleted")
	}
}

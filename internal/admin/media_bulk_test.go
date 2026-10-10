package admin_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/robertbaehr/GoPress/internal/admin"
)

func TestMediaListHasBulkActions(t *testing.T) {
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

	req := httptest.NewRequest(http.MethodGet, "/wp-admin/upload.php?mode=list", nil)
	req.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	rec := httptest.NewRecorder()
	rc := c.ForRequest(rec, req)
	if !(&admin.Handler{}).Handle(rec, req, rc, "/wp-admin/upload.php") {
		t.Fatal("upload.php not handled")
	}
	body := rec.Body.String()
	for _, want := range []string{
		`name="action"`,
		`value="delete"`,
		`Delete permanently`,
		`name="media[]"`,
		`id="posts-filter"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing %q in list HTML", want)
		}
	}
}

func TestMediaBulkDelete(t *testing.T) {
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

	now := time.Now().UTC().Format("2006-01-02 15:04:05")
	q, args, err := c.DB.Prepare(
		"INSERT INTO "+c.DB.Posts()+
			" (post_author, post_date, post_date_gmt, post_content, post_title, post_excerpt, post_status, "+
			"comment_status, ping_status, post_password, post_name, to_ping, pinged, post_modified, post_modified_gmt, "+
			"post_content_filtered, post_parent, guid, menu_order, post_type, post_mime_type, comment_count) VALUES "+
			"(%d, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %d, %s, %d, %s, %s, %d)",
		user.ID, now, now, "", "Bulk Delete Me", "", "inherit",
		"open", "closed", "", "bulk-delete-me", "", "", now, now,
		"", 0, "http://example.com/bulk-delete-me.jpg", 0, "attachment", "image/jpeg", 0,
	)
	if err != nil {
		t.Fatal(err)
	}
	res, err := c.DB.Exec(q, args...)
	if err != nil {
		t.Fatal(err)
	}
	id, err := res.LastInsertId()
	if err != nil || id <= 0 {
		t.Fatalf("insert id: %v", err)
	}
	t.Cleanup(func() {
		dq, dargs, _ := c.DB.Prepare("DELETE FROM "+c.DB.Posts()+" WHERE ID = %d", id)
		_, _ = c.DB.Exec(dq, dargs...)
	})

	form := url.Values{}
	form.Set("action", "delete")
	form.Add("media[]", strconv.FormatInt(id, 10))
	req := httptest.NewRequest(http.MethodPost, "/wp-admin/upload.php?mode=list", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	rec := httptest.NewRecorder()
	rc := c.ForRequest(rec, req)
	if !(&admin.Handler{}).Handle(rec, req, rc, "/wp-admin/upload.php") {
		t.Fatal("upload.php not handled")
	}
	if rec.Code != http.StatusFound {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	loc := rec.Header().Get("Location")
	if !strings.Contains(loc, "deleted=1") {
		t.Fatalf("Location=%q", loc)
	}

	cq, cargs, err := c.DB.Prepare(
		"SELECT ID FROM "+c.DB.Posts()+" WHERE ID = %d AND post_type = %s LIMIT 1",
		id, "attachment",
	)
	if err != nil {
		t.Fatal(err)
	}
	row, err := c.DB.GetRow(cq, cargs...)
	if err != nil {
		t.Fatal(err)
	}
	if row != nil {
		t.Fatalf("attachment %d still present", id)
	}
}

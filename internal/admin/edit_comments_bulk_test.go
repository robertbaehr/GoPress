package admin_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/admin"
)

func TestEditCommentsBulkApprove(t *testing.T) {
	c := bootOrSkip(t)
	user, err := c.GetUserBy("login", "admin")
	if err != nil || user == nil {
		t.Fatalf("admin: %v", err)
	}
	set, err := c.SetAuthCookie(user, false, false, 0, "")
	if err != nil {
		t.Fatalf("cookie: %v", err)
	}

	q, args, err := c.DB.Prepare(
		"INSERT INTO "+c.DB.Comments()+
			" (comment_post_ID, comment_author, comment_author_email, comment_author_url, comment_author_IP, "+
			"comment_date, comment_date_gmt, comment_content, comment_karma, comment_approved, comment_agent, "+
			"comment_type, comment_parent, user_id) VALUES "+
			"(%d, %s, %s, %s, %s, %s, %s, %s, %d, %s, %s, %s, %d, %d)",
		1, "Bulk A", "bulk@example.com", "", "127.0.0.1",
		"2024-01-01 00:00:00", "2024-01-01 00:00:00", "bulk approve me", 0, "0", "test",
		"comment", 0, 0,
	)
	if err != nil {
		t.Fatal(err)
	}
	res, err := c.DB.Exec(q, args...)
	if err != nil {
		t.Fatal(err)
	}
	cid, _ := res.LastInsertId()
	if cid <= 0 {
		t.Fatal("no insert id")
	}

	form := url.Values{}
	form.Set("action", "approve")
	form.Add("delete_comments[]", strconv.FormatInt(cid, 10))
	req := httptest.NewRequest(http.MethodPost, "/wp-admin/edit-comments.php", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	rec := httptest.NewRecorder()
	rc := c.ForRequest(rec, req)
	if !(&admin.Handler{}).Handle(rec, req, rc, "/wp-admin/edit-comments.php") {
		t.Fatal("not handled")
	}
	if rec.Code != http.StatusFound {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	loc := rec.Header().Get("Location")
	if !strings.Contains(loc, "approved=") {
		t.Fatalf("Location=%q", loc)
	}

	cq, cargs, err := c.DB.Prepare("SELECT comment_approved FROM "+c.DB.Comments()+" WHERE comment_ID = %d", cid)
	if err != nil {
		t.Fatal(err)
	}
	row, err := c.DB.GetRow(cq, cargs...)
	if err != nil || row == nil {
		t.Fatalf("reload: %v", err)
	}
	if got := fmt.Sprint(row["comment_approved"]); got != "1" {
		t.Fatalf("approved=%q", got)
	}
}

func TestEditCommentsListHasBulkUI(t *testing.T) {
	c := bootOrSkip(t)
	user, err := c.GetUserBy("login", "admin")
	if err != nil || user == nil {
		t.Fatalf("admin: %v", err)
	}
	set, err := c.SetAuthCookie(user, false, false, 0, "")
	if err != nil {
		t.Fatalf("cookie: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/wp-admin/edit-comments.php", nil)
	req.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	rec := httptest.NewRecorder()
	rc := c.ForRequest(rec, req)
	if !(&admin.Handler{}).Handle(rec, req, rc, "/wp-admin/edit-comments.php") {
		t.Fatal("not handled")
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d", rec.Code)
	}
	body := rec.Body.String()
	for _, needle := range []string{
		`name="action"`, `name="action2"`, `name="delete_comments[]"`,
		`Bulk actions`, `id="doaction"`, `the-comment-list`,
	} {
		if !strings.Contains(body, needle) {
			t.Fatalf("missing %q in %q", needle, trunc(body, 500))
		}
	}
}

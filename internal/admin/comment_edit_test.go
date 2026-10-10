package admin_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/admin"
)

func TestCommentModerateStub(t *testing.T) {
	c := bootOrSkip(t)
	user, err := c.GetUserBy("login", "admin")
	if err != nil || user == nil {
		t.Fatalf("admin: %v", err)
	}
	set, err := c.SetAuthCookie(user, false, false, 0, "")
	if err != nil {
		t.Fatalf("cookie: %v", err)
	}

	q, args, err := c.DB.Prepare("SELECT comment_ID FROM " + c.DB.Comments() + " ORDER BY comment_ID ASC LIMIT 1")
	if err != nil {
		t.Fatal(err)
	}
	row, err := c.DB.GetRow(q, args...)
	if err != nil || row == nil {
		t.Fatalf("need a fixture comment: %v", err)
	}
	cid := asInt64Any(row["comment_ID"])
	if cid <= 0 {
		t.Fatal("no comment id")
	}

	req := httptest.NewRequest(http.MethodGet, "/wp-admin/comment.php?action=editcomment&c="+strconv.FormatInt(cid, 10), nil)
	req.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	rec := httptest.NewRecorder()
	rc := c.ForRequest(rec, req)
	if !(&admin.Handler{}).Handle(rec, req, rc, "/wp-admin/comment.php") {
		t.Fatal("comment.php not handled")
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Edit Comment") && !strings.Contains(body, "Moderate Comment") {
		t.Fatalf("missing heading: %q", trunc(body, 400))
	}
	if !strings.Contains(body, "Approve") || !strings.Contains(body, "Spam") {
		t.Fatalf("expected moderation actions: %q", trunc(body, 600))
	}
}

func TestCommentApproveAction(t *testing.T) {
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
		1, "Mod Stub", "mod@example.com", "", "127.0.0.1",
		"2024-01-01 00:00:00", "2024-01-01 00:00:00", "pending for moderate stub", 0, "0", "test",
		"comment", 0, 0,
	)
	if err != nil {
		t.Fatal(err)
	}
	res, err := c.DB.Exec(q, args...)
	if err != nil {
		t.Fatal(err)
	}
	cid, err := res.LastInsertId()
	if err != nil || cid <= 0 {
		t.Fatalf("insert id: %v %d", err, cid)
	}
	t.Cleanup(func() {
		dq, dargs, err := c.DB.Prepare("DELETE FROM "+c.DB.Comments()+" WHERE comment_ID = %d", cid)
		if err == nil {
			_, _ = c.DB.Exec(dq, dargs...)
		}
	})

	req := httptest.NewRequest(http.MethodGet, "/wp-admin/comment.php?action=approvecomment&c="+strconv.FormatInt(cid, 10), nil)
	req.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	rec := httptest.NewRecorder()
	rc := c.ForRequest(rec, req)
	if !(&admin.Handler{}).Handle(rec, req, rc, "/wp-admin/comment.php") {
		t.Fatal("comment.php not handled")
	}
	if rec.Code != http.StatusFound {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	loc := rec.Header().Get("Location")
	if !strings.Contains(loc, "edit-comments.php") {
		t.Fatalf("redirect loc=%q", loc)
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
		t.Fatalf("approved=%q want 1", got)
	}
}

func asInt64Any(v any) int64 {
	switch t := v.(type) {
	case int64:
		return t
	case int:
		return int64(t)
	case []byte:
		n, _ := strconv.ParseInt(string(t), 10, 64)
		return n
	case string:
		n, _ := strconv.ParseInt(t, 10, 64)
		return n
	default:
		n, _ := strconv.ParseInt(fmt.Sprint(v), 10, 64)
		return n
	}
}

func trunc(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

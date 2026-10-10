package admin_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/admin"
)

func TestCommentReplyFormGET(t *testing.T) {
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

	req := httptest.NewRequest(http.MethodGet, "/wp-admin/comment.php?action=replyto&c="+strconv.FormatInt(cid, 10), nil)
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
	for _, want := range []string{
		"Reply to Comment",
		`id="replyto-comment"`,
		`name="content"`,
		`action=replytocomment`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing %q in %q", want, trunc(body, 800))
		}
	}
}

func TestCommentReplyFormPOST(t *testing.T) {
	c := bootOrSkip(t)
	user, err := c.GetUserBy("login", "admin")
	if err != nil || user == nil {
		t.Fatalf("admin: %v", err)
	}
	set, err := c.SetAuthCookie(user, false, false, 0, "")
	if err != nil {
		t.Fatalf("cookie: %v", err)
	}

	q, args, err := c.DB.Prepare("SELECT comment_ID, comment_post_ID FROM " + c.DB.Comments() + " ORDER BY comment_ID ASC LIMIT 1")
	if err != nil {
		t.Fatal(err)
	}
	row, err := c.DB.GetRow(q, args...)
	if err != nil || row == nil {
		t.Fatalf("need a fixture comment: %v", err)
	}
	parentID := asInt64Any(row["comment_ID"])
	postID := asInt64Any(row["comment_post_ID"])

	form := url.Values{}
	form.Set("action", "replytocomment")
	form.Set("content", "GoPress admin reply body")
	form.Set("comment_ID", strconv.FormatInt(parentID, 10))
	form.Set("comment_post_ID", strconv.FormatInt(postID, 10))

	req := httptest.NewRequest(http.MethodPost, "/wp-admin/comment.php?action=replytocomment&c="+strconv.FormatInt(parentID, 10), strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
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
	if !strings.Contains(loc, "edit-comments.php") || !strings.Contains(loc, "reply=1") {
		t.Fatalf("redirect loc=%q", loc)
	}

	cq, cargs, err := c.DB.Prepare(
		"SELECT comment_ID, comment_parent, comment_content FROM "+c.DB.Comments()+
			" WHERE comment_parent = %d AND comment_content = %s ORDER BY comment_ID DESC LIMIT 1",
		parentID, "GoPress admin reply body",
	)
	if err != nil {
		t.Fatal(err)
	}
	created, err := c.DB.GetRow(cq, cargs...)
	if err != nil || created == nil {
		t.Fatalf("reply not inserted: %v", err)
	}
	replyID := asInt64Any(created["comment_ID"])
	t.Cleanup(func() {
		dq, dargs, err := c.DB.Prepare("DELETE FROM "+c.DB.Comments()+" WHERE comment_ID = %d", replyID)
		if err == nil {
			_, _ = c.DB.Exec(dq, dargs...)
		}
	})
}

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
		`subsubsub`, `comment_status=moderated`, `comment_status=spam`,
		`value="approve"`, `value="spam"`, `value="trash"`,
	} {
		if !strings.Contains(body, needle) {
			t.Fatalf("missing %q in %q", needle, trunc(body, 500))
		}
	}
}

func TestEditCommentsSpamViewBulkUnspam(t *testing.T) {
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
		1, "Spam B", "spam@example.com", "", "127.0.0.1",
		"2024-01-02 00:00:00", "2024-01-02 00:00:00", "bulk unspam me", 0, "spam", "test",
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

	req := httptest.NewRequest(http.MethodGet, "/wp-admin/edit-comments.php?comment_status=spam", nil)
	req.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	rec := httptest.NewRecorder()
	rc := c.ForRequest(rec, req)
	if !(&admin.Handler{}).Handle(rec, req, rc, "/wp-admin/edit-comments.php") {
		t.Fatal("not handled")
	}
	body := rec.Body.String()
	if !strings.Contains(body, `value="unspam"`) || !strings.Contains(body, "Not spam") {
		t.Fatalf("spam view missing unspam: %q", trunc(body, 600))
	}
	if strings.Contains(body, `value="approve"`) {
		t.Fatalf("spam view should not list approve: %q", trunc(body, 600))
	}

	form := url.Values{}
	form.Set("action", "unspam")
	form.Set("comment_status", "spam")
	form.Add("delete_comments[]", strconv.FormatInt(cid, 10))
	post := httptest.NewRequest(http.MethodPost, "/wp-admin/edit-comments.php?comment_status=spam", strings.NewReader(form.Encode()))
	post.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	post.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	prec := httptest.NewRecorder()
	prc := c.ForRequest(prec, post)
	if !(&admin.Handler{}).Handle(prec, post, prc, "/wp-admin/edit-comments.php") {
		t.Fatal("post not handled")
	}
	if prec.Code != http.StatusFound {
		t.Fatalf("status=%d body=%s", prec.Code, prec.Body.String())
	}
	loc := prec.Header().Get("Location")
	if !strings.Contains(loc, "unspammed=") {
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

func TestEditCommentsSearchAndResponseColumns(t *testing.T) {
	// P7.107: search box + In response to + Submitted on columns.
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
		`class="search-box"`,
		`id="comment-search-input"`,
		`name="s"`,
		`column-response`,
		`In response to`,
		`column-date`,
		`Submitted on`,
		`comments-view-item-link`,
		`class="submitted-on"`,
	} {
		if !strings.Contains(body, needle) {
			t.Fatalf("missing %q", needle)
		}
	}

	req = httptest.NewRequest(http.MethodGet, "/wp-admin/edit-comments.php?s=zzznomatchcomment999", nil)
	req.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	rec = httptest.NewRecorder()
	rc = c.ForRequest(rec, req)
	if !(&admin.Handler{}).Handle(rec, req, rc, "/wp-admin/edit-comments.php") {
		t.Fatal("not handled")
	}
	body = rec.Body.String()
	if !strings.Contains(body, "No comments found") {
		t.Fatalf("expected empty search results: %q", trunc(body, 400))
	}
	if !strings.Contains(body, `value="zzznomatchcomment999"`) {
		t.Fatal("search input should retain query")
	}
}

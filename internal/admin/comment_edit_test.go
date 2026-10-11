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

func TestCommentEditFormGET(t *testing.T) {
	// P7.119: editcomment shows edit-form-comment fields.
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
		"SELECT comment_ID FROM "+c.DB.Comments()+" WHERE comment_approved = %s ORDER BY comment_ID ASC LIMIT 1",
		"1",
	)
	if err != nil {
		t.Fatal(err)
	}
	row, err := c.DB.GetRow(q, args...)
	if err != nil || row == nil {
		t.Fatalf("need an approved fixture comment: %v", err)
	}
	cid := asInt64Any(row["comment_ID"])

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
	for _, want := range []string{
		"Edit Comment",
		`name="newcomment_author"`,
		`name="newcomment_author_email"`,
		`name="newcomment_author_url"`,
		`name="content"`,
		`value="editedcomment"`,
		`id="post"`,
		`class="form-table editcomment"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing %q in body: %s", want, body[:min(800, len(body))])
		}
	}
}

func TestCommentEditFormPOST(t *testing.T) {
	// P7.119: editedcomment updates author/email/url/content.
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
		"SELECT comment_ID, comment_post_ID FROM "+c.DB.Comments()+" WHERE comment_approved = %s ORDER BY comment_ID ASC LIMIT 1",
		"1",
	)
	if err != nil {
		t.Fatal(err)
	}
	row, err := c.DB.GetRow(q, args...)
	if err != nil || row == nil {
		t.Fatalf("need an approved fixture comment: %v", err)
	}
	cid := asInt64Any(row["comment_ID"])
	postID := asInt64Any(row["comment_post_ID"])

	form := url.Values{}
	form.Set("action", "editedcomment")
	form.Set("comment_ID", strconv.FormatInt(cid, 10))
	form.Set("comment_post_ID", strconv.FormatInt(postID, 10))
	form.Set("newcomment_author", "Edited Author P7119")
	form.Set("newcomment_author_email", "edited-p7119@example.com")
	form.Set("newcomment_author_url", "https://example.com/p7119")
	form.Set("content", "Edited comment body for P7.119")

	req := httptest.NewRequest(http.MethodPost, "/wp-admin/comment.php", strings.NewReader(form.Encode()))
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
	if !strings.Contains(loc, "edit-comments.php") || !strings.Contains(loc, "#comment-"+strconv.FormatInt(cid, 10)) {
		t.Fatalf("redirect loc=%q", loc)
	}

	q2, args2, err := c.DB.Prepare(
		"SELECT comment_author, comment_author_email, comment_author_url, comment_content FROM "+c.DB.Comments()+" WHERE comment_ID = %d",
		cid,
	)
	if err != nil {
		t.Fatal(err)
	}
	row2, err := c.DB.GetRow(q2, args2...)
	if err != nil || row2 == nil {
		t.Fatalf("reload: %v", err)
	}
	if asStringAny(row2["comment_author"]) != "Edited Author P7119" {
		t.Fatalf("author=%v", row2["comment_author"])
	}
	if asStringAny(row2["comment_author_email"]) != "edited-p7119@example.com" {
		t.Fatalf("email=%v", row2["comment_author_email"])
	}
	if asStringAny(row2["comment_author_url"]) != "https://example.com/p7119" {
		t.Fatalf("url=%v", row2["comment_author_url"])
	}
	if asStringAny(row2["comment_content"]) != "Edited comment body for P7.119" {
		t.Fatalf("content=%v", row2["comment_content"])
	}
}

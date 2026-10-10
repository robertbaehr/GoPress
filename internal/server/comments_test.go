package server_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/server"
)

func TestFrontSingularRendersComments(t *testing.T) {
	c := testBoot(t)
	srv := &server.Server{WP: c, AbsPath: "/workspace/", Front: &server.Front{}}
	req := httptest.NewRequest(http.MethodGet, "/?p=1", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, `id="comments"`) {
		t.Fatalf("expected comments section, got %q", body)
	}
	if !strings.Contains(body, "A WordPress Commenter") {
		t.Fatalf("expected fixture comment author, got %q", body)
	}
	if !strings.Contains(body, `action="`) || !strings.Contains(body, `/wp-comments-post.php"`) {
		t.Fatalf("expected comment form, got %q", body)
	}
	if !strings.Contains(body, `id="commentform"`) || !strings.Contains(body, `wp-comment-cookies-consent`) {
		t.Fatalf("expected WP comment_form fields, got %q", body)
	}
	if !strings.Contains(body, `name="comment_post_ID"`) {
		t.Fatalf("expected comment_post_ID field, got %q", body)
	}
}

func TestCommentsPostCreatesHeldComment(t *testing.T) {
	c := testBoot(t)
	srv := &server.Server{WP: c, AbsPath: "/workspace/", Front: &server.Front{}}

	form := url.Values{}
	form.Set("comment_post_ID", "1")
	form.Set("author", "GoPress Tester")
	form.Set("email", "gopress-new-commenter@example.invalid")
	form.Set("url", "")
	form.Set("comment", "Hello from GoPress front comments test.")
	form.Set("comment_parent", "0")

	req := httptest.NewRequest(http.MethodPost, "/wp-comments-post.php", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusFound {
		t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String())
	}
	loc := rec.Header().Get("Location")
	if !strings.Contains(loc, "#comment-") {
		t.Fatalf("Location=%q", loc)
	}

	// Cleanup: find and delete the inserted comment.
	q, args, err := c.DB.Prepare(
		"SELECT comment_ID, comment_approved FROM "+c.DB.Comments()+
			" WHERE comment_author_email = %s AND comment_content = %s ORDER BY comment_ID DESC LIMIT 1",
		"gopress-new-commenter@example.invalid", "Hello from GoPress front comments test.",
	)
	if err != nil {
		t.Fatal(err)
	}
	row, err := c.DB.GetRow(q, args...)
	if err != nil || row == nil {
		t.Fatalf("inserted comment missing: %v", err)
	}
	approved := fmt.Sprint(row["comment_approved"])
	if approved != "0" {
		t.Fatalf("expected hold for new email, got approved=%q", approved)
	}
	id := row["comment_ID"]
	dq, dargs, err := c.DB.Prepare("DELETE FROM "+c.DB.Comments()+" WHERE comment_ID = %d", id)
	if err == nil {
		_, _ = c.DB.Exec(dq, dargs...)
	}
}

func TestCommentsPostMethodNotAllowed(t *testing.T) {
	c := testBoot(t)
	srv := &server.Server{WP: c, AbsPath: "/workspace/"}
	req := httptest.NewRequest(http.MethodGet, "/wp-comments-post.php", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status=%d", rec.Code)
	}
}

func TestCommentsPostLoggedInApproves(t *testing.T) {
	c := testBoot(t)
	user, err := c.GetUserBy("login", "admin")
	if err != nil || user == nil {
		t.Fatalf("admin: %v", err)
	}
	set, err := c.SetAuthCookie(user, false, false, 0, "")
	if err != nil {
		t.Fatalf("cookie: %v", err)
	}
	srv := &server.Server{WP: c, AbsPath: "/workspace/", Front: &server.Front{}}
	form := url.Values{}
	form.Set("comment_post_ID", "1")
	form.Set("comment", "Logged-in GoPress comment body.")
	form.Set("comment_parent", "0")
	req := httptest.NewRequest(http.MethodPost, "/wp-comments-post.php", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusFound {
		t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String())
	}

	q, args, err := c.DB.Prepare(
		"SELECT comment_ID, comment_approved, user_id FROM "+c.DB.Comments()+
			" WHERE comment_content = %s ORDER BY comment_ID DESC LIMIT 1",
		"Logged-in GoPress comment body.",
	)
	if err != nil {
		t.Fatal(err)
	}
	row, err := c.DB.GetRow(q, args...)
	if err != nil || row == nil {
		t.Fatalf("missing comment: %v", err)
	}
	if fmt.Sprint(row["comment_approved"]) != "1" {
		t.Fatalf("expected approved, got %v", row["comment_approved"])
	}
	if fmt.Sprint(row["user_id"]) != "1" {
		t.Fatalf("expected user_id=1, got %v", row["user_id"])
	}
	// Cleanup and restore comment_count.
	id := row["comment_ID"]
	dq, dargs, err := c.DB.Prepare("DELETE FROM "+c.DB.Comments()+" WHERE comment_ID = %d", id)
	if err == nil {
		_, _ = c.DB.Exec(dq, dargs...)
	}
	uq, uargs, err := c.DB.Prepare(
		"UPDATE "+c.DB.Posts()+" SET comment_count = GREATEST(comment_count - 1, 0) WHERE ID = %d",
		1,
	)
	if err == nil {
		_, _ = c.DB.Exec(uq, uargs...)
	}

}

package rest_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/robertbaehr/GoPress/internal/rest"
)

func TestCreateCommentRequiresAuth(t *testing.T) {
	c := bootOrSkip(t)
	t.Cleanup(func() { _ = c.DB.Close() })
	body := `{"post":1,"content":"Nope"}`
	req := httptest.NewRequest(http.MethodPost, "/wp-json/wp/v2/comments", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestCreateCommentAsLoggedInUser(t *testing.T) {
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
	ck := &http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue}

	createBody := `{"post":1,"content":"GoPress REST comment write"}`
	req := httptest.NewRequest(http.MethodPost, "/wp-json/wp/v2/comments", bytes.NewBufferString(createBody))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(ck)
	rec := httptest.NewRecorder()
	rc := c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, rc)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create status=%d body=%s", rec.Code, rec.Body.String())
	}
	var cm rest.Comment
	if err := json.Unmarshal(rec.Body.Bytes(), &cm); err != nil {
		t.Fatal(err)
	}
	if cm.ID <= 0 || cm.Post != 1 {
		t.Fatalf("created comment: %+v", cm)
	}
	if cm.Author != user.ID {
		t.Fatalf("expected author=%d got %d", user.ID, cm.Author)
	}
	if cm.Content.Rendered == "" {
		t.Fatalf("expected rendered content: %+v", cm)
	}
	commentID := cm.ID
	t.Cleanup(func() {
		q, args, err := c.DB.Prepare("DELETE FROM "+c.DB.Comments()+" WHERE comment_ID = %d", commentID)
		if err == nil {
			_, _ = c.DB.Exec(q, args...)
		}
		q, args, err = c.DB.Prepare(
			"UPDATE "+c.DB.Posts()+" SET comment_count = GREATEST(comment_count - 1, 0) WHERE ID = %d", 1,
		)
		if err == nil {
			_, _ = c.DB.Exec(q, args...)
		}
	})
}

func TestCreateCommentRequiresPost(t *testing.T) {
	c := bootOrSkip(t)
	t.Cleanup(func() { _ = c.DB.Close() })
	user, err := c.GetUserBy("login", "editor")
	if err != nil || user == nil {
		t.Fatalf("editor: %v", err)
	}
	set, err := c.SetAuthCookie(user, false, false, 0, "")
	if err != nil {
		t.Fatalf("cookie: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/wp-json/wp/v2/comments", bytes.NewBufferString(`{"content":"missing post"}`))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	rec := httptest.NewRecorder()
	rc := c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, rc)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

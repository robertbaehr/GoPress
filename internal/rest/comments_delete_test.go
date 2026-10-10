package rest_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/robertbaehr/GoPress/internal/rest"
	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestRESTCommentDeleteTrashAndForce(t *testing.T) {
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

	id, err := wp.InsertComment(c, wp.CommentInsert{
		PostID:   1,
		Author:   "Delete Me",
		Content:  "rest comment delete target",
		Approved: "1",
		UserID:   user.ID,
	})
	if err != nil || id <= 0 {
		t.Fatalf("insert: %v id=%d", err, id)
	}

	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/wp-json/wp/v2/comments/%d", id), nil)
	req.AddCookie(ck)
	rec := httptest.NewRecorder()
	rc := c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, rc)
	if rec.Code != http.StatusOK {
		t.Fatalf("trash status=%d body=%s", rec.Code, rec.Body.String())
	}
	var trashed map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &trashed); err != nil {
		t.Fatalf("json: %v", err)
	}
	if fmt.Sprint(trashed["status"]) != "trash" {
		t.Fatalf("expected trash status, got %#v", trashed)
	}

	again := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/wp-json/wp/v2/comments/%d", id), nil)
	again.AddCookie(ck)
	rec2 := httptest.NewRecorder()
	rc2 := c.ForRequest(rec2, again)
	(&rest.Server{}).Handle(rec2, again, rc2)
	if rec2.Code != http.StatusGone {
		t.Fatalf("already trashed status=%d body=%s", rec2.Code, rec2.Body.String())
	}

	force := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/wp-json/wp/v2/comments/%d?force=true", id), nil)
	force.AddCookie(ck)
	rec3 := httptest.NewRecorder()
	rc3 := c.ForRequest(rec3, force)
	(&rest.Server{}).Handle(rec3, force, rc3)
	if rec3.Code != http.StatusOK {
		t.Fatalf("force status=%d body=%s", rec3.Code, rec3.Body.String())
	}
	var deleted map[string]any
	if err := json.Unmarshal(rec3.Body.Bytes(), &deleted); err != nil {
		t.Fatalf("json: %v", err)
	}
	if deleted["deleted"] != true {
		t.Fatalf("expected deleted=true: %#v", deleted)
	}
	prev, _ := deleted["previous"].(map[string]any)
	if prev == nil || fmt.Sprint(prev["id"]) != fmt.Sprint(id) {
		t.Fatalf("previous missing: %#v", deleted)
	}

	gone := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/wp-json/wp/v2/comments/%d", id), nil)
	gone.AddCookie(ck)
	rec4 := httptest.NewRecorder()
	rc4 := c.ForRequest(rec4, gone)
	(&rest.Server{}).Handle(rec4, gone, rc4)
	if rec4.Code != http.StatusNotFound {
		t.Fatalf("get after delete status=%d body=%s", rec4.Code, rec4.Body.String())
	}
}

func TestRESTCommentDeleteRequiresAuth(t *testing.T) {
	c := bootOrSkip(t)
	t.Cleanup(func() { _ = c.DB.Close() })
	req := httptest.NewRequest(http.MethodDelete, "/wp-json/wp/v2/comments/1?force=true", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

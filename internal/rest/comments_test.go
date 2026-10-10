package rest_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/robertbaehr/GoPress/internal/rest"
)

func TestCommentsCollection(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/comments?per_page=5", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if rec.Header().Get("X-WP-Total") == "" {
		t.Fatal("expected X-WP-Total")
	}
	var comments []rest.Comment
	if err := json.Unmarshal(rec.Body.Bytes(), &comments); err != nil {
		t.Fatalf("json: %v body=%s", err, rec.Body.String())
	}
	if len(comments) == 0 || len(comments) > 5 {
		t.Fatalf("len=%d", len(comments))
	}
	cm := comments[0]
	if cm.ID <= 0 || cm.Post <= 0 || cm.Status != "approved" || cm.Type != "comment" {
		t.Fatalf("unexpected comment: %+v", cm)
	}
	if cm.Content.Rendered == "" || cm.Date == "" {
		t.Fatalf("missing fields: %+v", cm)
	}
}

func TestCommentsFilterByPost(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/comments?post=13&per_page=100", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var comments []rest.Comment
	if err := json.Unmarshal(rec.Body.Bytes(), &comments); err != nil {
		t.Fatalf("json: %v", err)
	}
	if len(comments) == 0 {
		t.Fatal("expected comments for post 13")
	}
	for _, cm := range comments {
		if cm.Post != 13 {
			t.Fatalf("post=%d want 13", cm.Post)
		}
		if cm.Status != "approved" {
			t.Fatalf("status=%q", cm.Status)
		}
		if cm.Type != "comment" {
			t.Fatalf("type=%q", cm.Type)
		}
	}
}

func TestCommentSingle(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/comments/1", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var cm rest.Comment
	if err := json.Unmarshal(rec.Body.Bytes(), &cm); err != nil {
		t.Fatalf("json: %v", err)
	}
	if cm.ID != 1 || cm.Post != 1 || cm.Status != "approved" {
		t.Fatalf("unexpected: %+v", cm)
	}
	if cm.AuthorName == "" || cm.Content.Rendered == "" {
		t.Fatalf("missing fields: %+v", cm)
	}
}

func TestCommentMissing(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/comments/999999991", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

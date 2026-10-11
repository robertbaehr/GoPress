package rest_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/robertbaehr/GoPress/internal/rest"
)

func TestCommentsOrderbyID(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/comments?per_page=1&orderby=id&order=desc", nil)
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
	if len(comments) != 1 {
		t.Fatalf("len=%d", len(comments))
	}
	descID := comments[0].ID

	req = httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/comments?per_page=1&orderby=id&order=asc", nil)
	rec = httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if err := json.Unmarshal(rec.Body.Bytes(), &comments); err != nil {
		t.Fatalf("json: %v", err)
	}
	if len(comments) != 1 {
		t.Fatalf("len=%d", len(comments))
	}
	ascID := comments[0].ID
	if ascID >= descID {
		t.Fatalf("orderby=id asc=%d desc=%d", ascID, descID)
	}
}

func TestCommentsSearch(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/comments?search=a&per_page=100", nil)
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
		t.Fatal("expected search hits")
	}
	if rec.Header().Get("X-WP-Total") == "" {
		t.Fatal("expected X-WP-Total")
	}
}

func TestCommentsParentFilter(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/comments?parent=0&per_page=100", nil)
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
		t.Fatal("expected top-level comments")
	}
	for _, cm := range comments {
		if cm.Parent != 0 {
			t.Fatalf("parent=%d want 0 for id=%d", cm.Parent, cm.ID)
		}
	}
}

func TestCommentsIncludeExclude(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/comments?include=1&per_page=10", nil)
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
	if len(comments) != 1 || comments[0].ID != 1 {
		t.Fatalf("include=1 got %#v", comments)
	}

	req = httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/comments?exclude=1&per_page=100", nil)
	rec = httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if err := json.Unmarshal(rec.Body.Bytes(), &comments); err != nil {
		t.Fatalf("json: %v", err)
	}
	for _, cm := range comments {
		if cm.ID == 1 {
			t.Fatal("exclude=1 still returned id=1")
		}
	}
}

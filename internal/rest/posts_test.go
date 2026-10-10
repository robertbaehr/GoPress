package rest_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/robertbaehr/GoPress/internal/rest"
)

func TestPostsListAnon(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/posts?per_page=5", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var posts []rest.Post
	if err := json.Unmarshal(rec.Body.Bytes(), &posts); err != nil {
		t.Fatalf("json array: %v body=%s", err, rec.Body.String())
	}
	if len(posts) == 0 || len(posts) > 5 {
		t.Fatalf("len=%d", len(posts))
	}
	p := posts[0]
	if p.ID <= 0 || p.Slug == "" || p.Status != "publish" || p.Type != "post" {
		t.Fatalf("unexpected post: %+v", p)
	}
	if p.Title.Rendered == "" {
		t.Fatal("expected title.rendered")
	}
	if p.Date == "" || !containsT(p.Date) {
		t.Fatalf("date=%q", p.Date)
	}
	if p.Link == "" {
		t.Fatal("expected link")
	}
}

func TestPostSingle(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/posts/1", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var post rest.Post
	if err := json.Unmarshal(rec.Body.Bytes(), &post); err != nil {
		t.Fatalf("json: %v body=%s", err, rec.Body.String())
	}
	if post.ID != 1 || post.Type != "post" || post.Slug != "hello-world" {
		t.Fatalf("unexpected post: %+v", post)
	}
	if post.Title.Rendered == "" || post.Link == "" || post.Date == "" {
		t.Fatalf("missing fields: %+v", post)
	}
	if !containsT(post.Date) {
		t.Fatalf("date=%q", post.Date)
	}
}

func TestPostSingleMissing(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/posts/999999991", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func containsT(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] == 'T' {
			return true
		}
	}
	return false
}

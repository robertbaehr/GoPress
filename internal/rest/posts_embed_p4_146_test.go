package rest_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/robertbaehr/GoPress/internal/rest"
)

func TestPostsEmbedQueryAttachesAuthor(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/posts/1?_embed=1", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	embedded, ok := body["_embedded"].(map[string]any)
	if !ok {
		t.Fatalf("expected _embedded: keys=%v", mapKeys(body))
	}
	authors, ok := embedded["author"].([]any)
	if !ok || len(authors) != 1 {
		t.Fatalf("expected author embed: %#v", embedded["author"])
	}
	author, ok := authors[0].(map[string]any)
	if !ok {
		t.Fatalf("author[0] type: %#v", authors[0])
	}
	if author["id"] == nil || author["name"] == nil || author["slug"] == nil {
		t.Fatalf("author embed missing fields: %#v", author)
	}
	if _, has := body["content"]; !has {
		t.Fatal("view context should still include content with _embed")
	}
}

func TestPostsEmbedQueryWithoutParamOmitsEmbedded(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/posts/1", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if _, ok := body["_embedded"]; ok {
		t.Fatalf("unexpected _embedded without ?_embed: keys=%v", mapKeys(body))
	}
}

func TestPostsEmbedRelationFilterAuthorOnly(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/posts/1?_embed=author", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	embedded, ok := body["_embedded"].(map[string]any)
	if !ok {
		t.Fatalf("expected _embedded: %#v", body["_embedded"])
	}
	if _, ok := embedded["author"]; !ok {
		t.Fatalf("expected author: keys=%v", mapKeys(embedded))
	}
	if _, ok := embedded["wp:term"]; ok {
		t.Fatalf("author-only filter should omit wp:term: keys=%v", mapKeys(embedded))
	}
	if _, ok := embedded["wp:featuredmedia"]; ok {
		t.Fatalf("author-only filter should omit featuredmedia: keys=%v", mapKeys(embedded))
	}
}

func TestPostsCollectionEmbedAttachesPerItem(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/posts?per_page=1&_embed", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var body []map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body) == 0 {
		t.Fatal("expected at least one post")
	}
	embedded, ok := body[0]["_embedded"].(map[string]any)
	if !ok {
		t.Fatalf("collection item missing _embedded: %#v", body[0])
	}
	if _, ok := embedded["author"]; !ok {
		t.Fatalf("collection embed missing author: keys=%v", mapKeys(embedded))
	}
}

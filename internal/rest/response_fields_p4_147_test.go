package rest_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/robertbaehr/GoPress/internal/rest"
)

func TestPostsFieldsSparseProjection(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/posts?per_page=1&_fields=id,title", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var rows []map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &rows); err != nil {
		t.Fatal(err)
	}
	if len(rows) == 0 {
		t.Fatal("expected at least one post")
	}
	for k := range rows[0] {
		if k != "id" && k != "title" {
			t.Fatalf("unexpected key %q in %v", k, rows[0])
		}
	}
}

func TestPostItemFieldsNestedTitle(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/posts/1?_fields=id,title.rendered", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var row map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &row); err != nil {
		t.Fatal(err)
	}
	if len(row) != 2 {
		t.Fatalf("keys=%v", row)
	}
	title, ok := row["title"].(map[string]any)
	if !ok {
		t.Fatalf("title=%T %#v", row["title"], row["title"])
	}
	if _, ok := title["rendered"]; !ok {
		t.Fatalf("title=%v", title)
	}
	if _, ok := title["raw"]; ok {
		t.Fatalf("raw should be omitted: %v", title)
	}
}

func TestPostItemFieldsLinksOptIn(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/posts/1?_fields=id,_links", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var row map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &row); err != nil {
		t.Fatal(err)
	}
	if _, ok := row["_links"]; !ok {
		t.Fatalf("expected _links: %v", row)
	}
	if _, ok := row["title"]; ok {
		t.Fatalf("title should be omitted: %v", row)
	}
}

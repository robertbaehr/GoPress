package rest_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/robertbaehr/GoPress/internal/rest"
	"github.com/robertbaehr/GoPress/internal/wp"
)

// P4.161: categories[terms]=2&include_children=true expands Fixture Root → Leaf posts.
func TestPostsCategoriesIncludeChildrenP4161(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	// Simple categories=2 → zero hits (Root has no direct posts).
	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/posts?categories=2&per_page=100&_fields=id", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("categories=2 status=%d body=%s", rec.Code, rec.Body.String())
	}
	var empty []map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &empty); err != nil {
		t.Fatal(err)
	}
	if len(empty) != 0 {
		t.Fatalf("expected no direct Root posts, got %d", len(empty))
	}

	// Object form with include_children → Leaf descendants.
	req = httptest.NewRequest(http.MethodGet,
		"/wp-json/wp/v2/posts?categories[terms]=2&categories[include_children]=true&per_page=100&_fields=id", nil)
	rec = httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("include_children status=%d body=%s", rec.Code, rec.Body.String())
	}
	var hits []map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &hits); err != nil {
		t.Fatal(err)
	}
	if len(hits) < 7 {
		t.Fatalf("expected ≥7 child posts under Root, got %d: %v", len(hits), hits)
	}
}

// P4.161: object form without include_children defaults false (no expansion).
func TestPostsCategoriesObjectNoChildrenP4161(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()
	req := httptest.NewRequest(http.MethodGet,
		"/wp-json/wp/v2/posts?categories[terms]=2&per_page=100&_fields=id", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var hits []map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &hits); err != nil {
		t.Fatal(err)
	}
	if len(hits) != 0 {
		t.Fatalf("object form without include_children should not expand: %v", hits)
	}
}

// P4.161: categories_exclude object form with include_children excludes leaf posts.
func TestPostsCategoriesExcludeIncludeChildrenP4161(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()
	req := httptest.NewRequest(http.MethodGet,
		"/wp-json/wp/v2/posts?categories_exclude[terms]=2&categories_exclude[include_children]=true&per_page=5&_fields=id", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var hits []map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &hits); err != nil {
		t.Fatal(err)
	}
	if len(hits) == 0 {
		t.Fatal("expected non-leaf posts when excluding Root+children")
	}
	for _, p := range hits {
		id, _ := p["id"].(float64)
		if int64(id) == 13 {
			t.Fatalf("excluded leaf post 13 still present: %v", hits)
		}
	}
}

// P4.161: GetTermChildren walks Fixture Root → Mid → Leaf.
func TestGetTermChildrenP4161(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()
	kids := wp.GetTermChildren(c, 2, "category")
	seen := map[int64]bool{}
	for _, id := range kids {
		seen[id] = true
	}
	if !seen[3] || !seen[4] {
		t.Fatalf("expected Mid(3)+Leaf(4) under Root(2), got %v", kids)
	}
}

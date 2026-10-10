package rest_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/rest"
)

func TestSearchHello(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/search?search=hello", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if rec.Header().Get("X-WP-Total") == "" {
		t.Fatal("expected X-WP-Total")
	}
	var results []rest.SearchResult
	if err := json.Unmarshal(rec.Body.Bytes(), &results); err != nil {
		t.Fatalf("json: %v body=%s", err, rec.Body.String())
	}
	if len(results) == 0 {
		t.Fatal("expected search hits")
	}
	found := false
	for _, r := range results {
		idMatch := false
		switch id := r.ID.(type) {
		case float64:
			idMatch = int64(id) == 1
		case int64:
			idMatch = id == 1
		case json.Number:
			n, _ := id.Int64()
			idMatch = n == 1
		}
		if idMatch || strings.Contains(strings.ToLower(r.Title), "hello") {
			found = true
			if r.Type != "post" || r.Subtype == "" || r.URL == "" || r.Title == "" {
				t.Fatalf("unexpected result: %+v", r)
			}
			if r.Links == nil || r.Links["self"] == nil || r.Links["collection"] == nil || r.Links["about"] == nil {
				t.Fatalf("expected _links: %+v", r.Links)
			}
		}
	}
	if !found {
		t.Fatalf("expected hello-world hit; got %+v", results)
	}
}

func TestSearchEmptyTerm(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/search?search=", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var results []rest.SearchResult
	if err := json.Unmarshal(rec.Body.Bytes(), &results); err != nil {
		t.Fatalf("json: %v", err)
	}
	// Empty search still returns published posts/pages (LIKE '%%').
	if results == nil {
		t.Fatal("expected array")
	}
}

func TestSearchTypeTerm(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/search?search=a&type=term", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var results []rest.SearchResult
	if err := json.Unmarshal(rec.Body.Bytes(), &results); err != nil {
		t.Fatalf("json: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected term hits")
	}
	for _, r := range results {
		if r.Type != "category" && r.Type != "post_tag" {
			t.Fatalf("unexpected type %q in %+v", r.Type, r)
		}
		if r.Subtype != "" {
			t.Fatalf("term results should omit subtype: %+v", r)
		}
		if r.URL == "" || r.Title == "" {
			t.Fatalf("incomplete result: %+v", r)
		}
	}
}

func TestSearchTypeTermSubtypeCategory(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/search?search=a&type=term&subtype=category", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var results []rest.SearchResult
	if err := json.Unmarshal(rec.Body.Bytes(), &results); err != nil {
		t.Fatalf("json: %v", err)
	}
	for _, r := range results {
		if r.Type != "category" {
			t.Fatalf("expected only categories, got %+v", r)
		}
	}
}

func TestSearchTypePostFormat(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/search?type=post-format&per_page=20", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var results []rest.SearchResult
	if err := json.Unmarshal(rec.Body.Bytes(), &results); err != nil {
		t.Fatalf("json: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected post-format hits")
	}
	foundAside := false
	for _, r := range results {
		if r.Type != "post-format" {
			t.Fatalf("type=%q", r.Type)
		}
		id, _ := r.ID.(string)
		if id == "" {
			t.Fatalf("expected string id: %+v", r)
		}
		if !strings.Contains(r.URL, "/type/") {
			t.Fatalf("url=%q", r.URL)
		}
		if id == "aside" {
			foundAside = true
		}
	}
	if !foundAside {
		t.Fatalf("expected aside format: %+v", results)
	}
}

func TestSearchTypeInvalid(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/search?type=bogus", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "rest_invalid_param") {
		t.Fatalf("body=%s", rec.Body.String())
	}
}

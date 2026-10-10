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

func TestSearchSubtypePage(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/search?search=&subtype=page&per_page=20", nil)
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
		t.Fatal("expected page hits")
	}
	for _, r := range results {
		if r.Subtype != "page" {
			t.Fatalf("expected subtype=page, got %+v", r)
		}
	}
}

func TestSearchSubtypePost(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/search?search=hello&subtype=post", nil)
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
		t.Fatal("expected post hits")
	}
	for _, r := range results {
		if r.Subtype != "post" {
			t.Fatalf("expected subtype=post, got %+v", r)
		}
	}
}

func TestSearchIncludeExclude(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/search?search=&include=1&per_page=20", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("include status=%d body=%s", rec.Code, rec.Body.String())
	}
	var results []rest.SearchResult
	if err := json.Unmarshal(rec.Body.Bytes(), &results); err != nil {
		t.Fatalf("json: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("include=1 want 1 got %d: %+v", len(results), results)
	}
	switch id := results[0].ID.(type) {
	case float64:
		if int64(id) != 1 {
			t.Fatalf("id=%v", id)
		}
	case int64:
		if id != 1 {
			t.Fatalf("id=%v", id)
		}
	default:
		t.Fatalf("id type %T", results[0].ID)
	}

	req = httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/search?search=hello&exclude=1&per_page=20", nil)
	rec = httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("exclude status=%d body=%s", rec.Code, rec.Body.String())
	}
	results = nil
	if err := json.Unmarshal(rec.Body.Bytes(), &results); err != nil {
		t.Fatalf("json: %v", err)
	}
	for _, r := range results {
		switch id := r.ID.(type) {
		case float64:
			if int64(id) == 1 {
				t.Fatalf("exclude=1 still returned id 1")
			}
		case int64:
			if id == 1 {
				t.Fatalf("exclude=1 still returned id 1")
			}
		}
	}
}

func TestSearchTermInclude(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	// Uncategorized is typically term_id 1.
	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/search?type=term&include=1&per_page=10", nil)
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
	if len(results) != 1 {
		t.Fatalf("want 1 term, got %d: %+v", len(results), results)
	}
}

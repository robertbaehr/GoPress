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
		if r.ID == 1 || strings.Contains(strings.ToLower(r.Title), "hello") {
			found = true
			if r.Type != "post" || r.Subtype == "" || r.URL == "" || r.Title == "" {
				t.Fatalf("unexpected result: %+v", r)
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

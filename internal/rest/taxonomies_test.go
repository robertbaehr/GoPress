package rest_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/rest"
)

func TestTaxonomiesList(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/taxonomies", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var tax map[string]rest.Taxonomy
	if err := json.Unmarshal(rec.Body.Bytes(), &tax); err != nil {
		t.Fatalf("json: %v body=%s", err, rec.Body.String())
	}
	for _, slug := range []string{"category", "post_tag"} {
		tx, ok := tax[slug]
		if !ok {
			t.Fatalf("missing taxonomy %q", slug)
		}
		if tx.Slug != slug || tx.RestNamespace != "wp/v2" || tx.RestBase == "" || tx.Name == "" {
			t.Fatalf("unexpected taxonomy: %+v", tx)
		}
	}
	if !tax["category"].Hierarchical {
		t.Fatal("category should be hierarchical")
	}
	if tax["post_tag"].Hierarchical {
		t.Fatal("post_tag should not be hierarchical")
	}
	if tax["category"].RestBase != "categories" || tax["post_tag"].RestBase != "tags" {
		t.Fatalf("rest_base category=%q tag=%q", tax["category"].RestBase, tax["post_tag"].RestBase)
	}
}

func TestCategoriesCollection(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/categories?per_page=2", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if rec.Header().Get("X-WP-Total") == "" {
		t.Fatal("expected X-WP-Total")
	}
	var terms []rest.Term
	if err := json.Unmarshal(rec.Body.Bytes(), &terms); err != nil {
		t.Fatalf("json: %v body=%s", err, rec.Body.String())
	}
	if len(terms) == 0 || len(terms) > 2 {
		t.Fatalf("len=%d", len(terms))
	}
	if terms[0].ID <= 0 || terms[0].Slug == "" || terms[0].Taxonomy != "category" || terms[0].Name == "" {
		t.Fatalf("unexpected term: %+v", terms[0])
	}
	if terms[0].Link == "" {
		t.Fatal("expected link")
	}
	if terms[0].Links == nil || terms[0].Links["self"] == nil || terms[0].Links["wp:post_type"] == nil {
		t.Fatalf("expected _links: %+v", terms[0].Links)
	}
	if terms[0].Meta == nil {
		t.Fatal("expected meta")
	}
}

func TestCategorySingle(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/categories/1", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var term rest.Term
	if err := json.Unmarshal(rec.Body.Bytes(), &term); err != nil {
		t.Fatalf("json: %v body=%s", err, rec.Body.String())
	}
	if term.ID != 1 || term.Slug != "uncategorized" || term.Taxonomy != "category" {
		t.Fatalf("unexpected term: %+v", term)
	}

	req = httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/categories/4", nil)
	rec = httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("leaf status=%d body=%s", rec.Code, rec.Body.String())
	}
	var leaf rest.Term
	if err := json.Unmarshal(rec.Body.Bytes(), &leaf); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(leaf.Link, "/category/fixture-root/fixture-mid/fixture-leaf/") {
		t.Fatalf("expected hierarchical category link, got %q", leaf.Link)
	}
	if leaf.Links["up"] == nil {
		t.Fatal("expected up link for child category")
	}
}

func TestTagsCollectionAndSingle(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/tags?per_page=3", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if rec.Header().Get("X-WP-Total") == "" {
		t.Fatal("expected X-WP-Total")
	}
	var terms []rest.Term
	if err := json.Unmarshal(rec.Body.Bytes(), &terms); err != nil {
		t.Fatalf("json: %v", err)
	}
	if len(terms) == 0 || len(terms) > 3 {
		t.Fatalf("len=%d", len(terms))
	}
	if terms[0].Taxonomy != "post_tag" {
		t.Fatalf("taxonomy=%q", terms[0].Taxonomy)
	}

	id := terms[0].ID
	req = httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/tags/"+strconv.FormatInt(id, 10), nil)
	rec = httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("single status=%d body=%s", rec.Code, rec.Body.String())
	}
	var one rest.Term
	if err := json.Unmarshal(rec.Body.Bytes(), &one); err != nil {
		t.Fatal(err)
	}
	if one.ID != id || one.Taxonomy != "post_tag" {
		t.Fatalf("unexpected: %+v", one)
	}
	if one.Links == nil || one.Links["collection"] == nil || one.Links["wp:post_type"] == nil {
		t.Fatalf("expected tag _links: %+v", one.Links)
	}
}

func TestTermMissing(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/categories/999999991", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

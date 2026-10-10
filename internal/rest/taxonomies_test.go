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
		if tx.Links == nil || tx.Links["collection"] == nil || tx.Links["wp:items"] == nil {
			t.Fatalf("expected taxonomy _links: %+v", tx.Links)
		}
		if tx.Visibility != nil {
			t.Fatal("view context should omit visibility")
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

func TestTaxonomySingleAndEditVisibility(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/taxonomies/category", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var tx rest.Taxonomy
	if err := json.Unmarshal(rec.Body.Bytes(), &tx); err != nil {
		t.Fatalf("json: %v", err)
	}
	if tx.Slug != "category" || tx.RestBase != "categories" || tx.Links == nil || tx.Links["wp:items"] == nil {
		t.Fatalf("unexpected: %+v", tx)
	}

	req = httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/taxonomies/nope", nil)
	rec = httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("invalid status=%d", rec.Code)
	}

	user, err := c.GetUserBy("login", "admin")
	if err != nil || user == nil {
		t.Skipf("admin unavailable: %v", err)
	}
	set, err := c.SetAuthCookie(user, false, false, 0, "")
	if err != nil {
		t.Fatalf("SetAuthCookie: %v", err)
	}
	req = httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/taxonomies/post_tag?context=edit", nil)
	req.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	rec = httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	c.AuthenticateRequest()
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("edit status=%d body=%s", rec.Code, rec.Body.String())
	}
	var edit rest.Taxonomy
	if err := json.Unmarshal(rec.Body.Bytes(), &edit); err != nil {
		t.Fatal(err)
	}
	if edit.Visibility == nil || !edit.Visibility.Public || !edit.Visibility.ShowUI {
		t.Fatalf("expected edit visibility: %+v", edit.Visibility)
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

func TestCategoriesCollectionFilters(t *testing.T) {
	// P4.120: search, slug, parent, hide_empty, post filters.
	c := bootOrSkip(t)
	defer c.DB.Close()

	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/categories?search=uncategorized", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("search status=%d body=%s", rec.Code, rec.Body.String())
	}
	var terms []rest.Term
	if err := json.Unmarshal(rec.Body.Bytes(), &terms); err != nil {
		t.Fatal(err)
	}
	if len(terms) == 0 || terms[0].Slug != "uncategorized" {
		t.Fatalf("search terms=%+v", terms)
	}

	req = httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/categories?slug=uncategorized", nil)
	rec = httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if err := json.Unmarshal(rec.Body.Bytes(), &terms); err != nil {
		t.Fatal(err)
	}
	if len(terms) != 1 || terms[0].Slug != "uncategorized" {
		t.Fatalf("slug filter: %+v", terms)
	}

	req = httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/categories?parent=0&per_page=100", nil)
	rec = httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if err := json.Unmarshal(rec.Body.Bytes(), &terms); err != nil {
		t.Fatal(err)
	}
	if len(terms) == 0 {
		t.Fatal("expected top-level categories")
	}
	for _, term := range terms {
		if term.Parent == nil || *term.Parent != 0 {
			t.Fatalf("parent=0 leaked %+v", term)
		}
	}

	req = httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/categories?hide_empty=true&per_page=100", nil)
	rec = httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if err := json.Unmarshal(rec.Body.Bytes(), &terms); err != nil {
		t.Fatal(err)
	}
	for _, term := range terms {
		if term.Count <= 0 {
			t.Fatalf("hide_empty leaked empty term %+v", term)
		}
	}

	req = httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/categories?post=1", nil)
	rec = httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("post filter status=%d body=%s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/categories?post=0", nil)
	rec = httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("invalid post status=%d body=%s", rec.Code, rec.Body.String())
	}
}

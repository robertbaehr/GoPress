package rest_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/rest"
)

// P4.160: orderby[]=slug → rest_invalid_param / rest_invalid_type (string schema).
func TestPostsOrderbySlugArrayFormInvalidP4160(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/posts?orderby[]=slug&per_page=1", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, "rest_invalid_type") || !strings.Contains(body, "orderby is not of type string") {
		t.Fatalf("expected string type error: %s", body)
	}
}

// P4.160: orderby=name is not in the REST enum (alias lives only in WP_Query).
func TestPostsOrderbyNameRejectedP4160(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/posts?orderby=name&per_page=1", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, "rest_invalid_param") || !strings.Contains(body, "orderby") {
		t.Fatalf("expected orderby enum reject: %s", body)
	}
	if !strings.Contains(body, "slug") {
		t.Fatalf("enum message should list slug: %s", body)
	}
}

// P4.160: orderby=slug&order=asc|desc sorts by post_name.
func TestPostsOrderbySlugAscDescP4160(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	include := "1,13,19"
	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/posts?orderby=slug&order=asc&include="+include+"&per_page=10", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("asc status=%d body=%s", rec.Code, rec.Body.String())
	}
	var asc []rest.Post
	if err := json.Unmarshal(rec.Body.Bytes(), &asc); err != nil {
		t.Fatal(err)
	}
	if len(asc) != 3 {
		t.Fatalf("asc len=%d want 3 body=%s", len(asc), rec.Body.String())
	}
	// fixture-post-01 < fixture-post-07 < hello-world
	got := fmt.Sprintf("%d,%d,%d", asc[0].ID, asc[1].ID, asc[2].ID)
	if got != "13,19,1" {
		t.Fatalf("asc ids=%s want 13,19,1 slugs=%v", got, []string{asc[0].Slug, asc[1].Slug, asc[2].Slug})
	}
	for i := 1; i < len(asc); i++ {
		if asc[i-1].Slug > asc[i].Slug {
			t.Fatalf("asc slug order broken: %q then %q", asc[i-1].Slug, asc[i].Slug)
		}
	}

	req = httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/posts?orderby=slug&order=desc&include="+include+"&per_page=10", nil)
	rec = httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("desc status=%d body=%s", rec.Code, rec.Body.String())
	}
	var desc []rest.Post
	if err := json.Unmarshal(rec.Body.Bytes(), &desc); err != nil {
		t.Fatal(err)
	}
	got = fmt.Sprintf("%d,%d,%d", desc[0].ID, desc[1].ID, desc[2].ID)
	if got != "1,19,13" {
		t.Fatalf("desc ids=%s want 1,19,13", got)
	}
}

// P4.160: pages also accept orderby=slug.
func TestPagesOrderbySlugAscP4160(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/pages?orderby=slug&order=asc&per_page=5", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var pages []rest.Post
	if err := json.Unmarshal(rec.Body.Bytes(), &pages); err != nil {
		t.Fatal(err)
	}
	if len(pages) < 2 {
		t.Fatalf("expected pages, got %d", len(pages))
	}
	for i := 1; i < len(pages); i++ {
		if pages[i-1].Slug > pages[i].Slug {
			t.Fatalf("page slug order broken: %q then %q", pages[i-1].Slug, pages[i].Slug)
		}
	}
}

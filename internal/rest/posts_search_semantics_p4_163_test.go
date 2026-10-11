package rest_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/rest"
)

// P4.163: REST ?search= multi-word AND + exclusion via query layer.
func TestPostsSearchMultiWordANDP4163(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	// Three-term AND: Fixture + Post + 01 → narrows to Fixture Post 01 family.
	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/posts?search=Fixture+Post+01&per_page=20", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var posts []rest.Post
	if err := json.Unmarshal(rec.Body.Bytes(), &posts); err != nil {
		t.Fatal(err)
	}
	if len(posts) == 0 {
		t.Fatal("expected Fixture Post 01 hits")
	}
	found := false
	for _, p := range posts {
		if strings.Contains(p.Title.Rendered, "Fixture Post 01") || p.Slug == "fixture-post-01" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected Fixture Post 01 among AND hits, got %d posts first=%q", len(posts), posts[0].Title.Rendered)
	}
}

func TestPostsSearchExclusionP4163(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/posts?search=Fixture+-accordion&per_page=50", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var posts []rest.Post
	if err := json.Unmarshal(rec.Body.Bytes(), &posts); err != nil {
		t.Fatal(err)
	}
	if len(posts) == 0 {
		t.Fatal("expected hits")
	}
	for _, p := range posts {
		hay := strings.ToLower(p.Title.Rendered)
		if strings.Contains(hay, "accordion") {
			t.Fatalf("exclusion failed: %q", p.Title.Rendered)
		}
	}
}

func TestPostsSearchQuotedPhraseP4163(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	req := httptest.NewRequest(http.MethodGet, `/wp-json/wp/v2/posts?search=%22Hello+world%22&per_page=10`, nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var posts []rest.Post
	if err := json.Unmarshal(rec.Body.Bytes(), &posts); err != nil {
		t.Fatal(err)
	}
	if len(posts) == 0 {
		t.Fatal("expected quoted phrase hits")
	}
}

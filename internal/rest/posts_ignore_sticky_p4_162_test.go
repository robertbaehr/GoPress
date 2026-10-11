package rest_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/rest"
)

// P4.162: ignore_sticky[]= is rest_invalid_type; 0/1 accepted; pages ignore param.
func TestPostsIgnoreStickyArrayInvalidP4162(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/posts?ignore_sticky[]=false&per_page=1", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, "rest_invalid_param") || !strings.Contains(body, "ignore_sticky") {
		t.Fatalf("expected ignore_sticky invalid type: %s", body)
	}
}

func TestPostsIgnoreStickyNumericZeroPrependsP4162(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	// ignore_sticky=0 → false → StickyFirst (prepend sticky 13).
	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/posts?ignore_sticky=0&per_page=5", nil)
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
	if len(posts) == 0 || posts[0].ID != 13 || !posts[0].Sticky {
		t.Fatalf("ignore_sticky=0 want sticky first, got %+v", posts)
	}

	// ignore_sticky=1 → true → no forced StickyFirst (still 200).
	req = httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/posts?ignore_sticky=1&per_page=5", nil)
	rec = httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("ignore_sticky=1 status=%d", rec.Code)
	}
}

func TestPostsIgnoreStickyFalseWithIncludeSkipsPrependP4162(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	// When include= sets post__in, WP unsets ignore_sticky_posts — no StickyFirst.
	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/posts?ignore_sticky=false&include=1,2&orderby=include&per_page=5", nil)
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
		t.Fatal("expected included posts")
	}
	// Sticky 13 must not be prepended when not in include.
	for _, p := range posts {
		if p.ID == 13 {
			t.Fatalf("sticky prepended despite include=: %+v", posts)
		}
	}
	if posts[0].ID != 1 {
		t.Fatalf("want include order starting at 1, got %+v", posts)
	}
}

func TestPagesIgnoreStickyParamIgnoredP4162(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	// Pages schema has no ignore_sticky — invalid values are ignored (not 400).
	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/pages?ignore_sticky=maybe&per_page=1", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("pages ignore_sticky=maybe should be ignored, status=%d body=%s", rec.Code, rec.Body.String())
	}
}

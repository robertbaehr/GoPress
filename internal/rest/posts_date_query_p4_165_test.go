package rest_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/rest"
)

// P4.165: after[]= / before[]= / modified_*[]= → rest_invalid_param / rest_invalid_type (string schema).
func TestPostsDateQueryArrayFormInvalidP4165(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	for _, tc := range []struct {
		path string
		key  string
	}{
		{"/wp-json/wp/v2/posts?after[]=2024-01-01T00:00:00&per_page=1", "after"},
		{"/wp-json/wp/v2/posts?before[]=2024-12-31T23:59:59&per_page=1", "before"},
		{"/wp-json/wp/v2/posts?modified_after[]=2024-01-01T00:00:00&per_page=1", "modified_after"},
		{"/wp-json/wp/v2/posts?modified_before[]=2024-12-31T23:59:59&per_page=1", "modified_before"},
		{"/wp-json/wp/v2/posts?after[0]=2024-01-01T00:00:00&per_page=1", "after"},
		{"/wp-json/wp/v2/pages?before[0]=2024-06-01T00:00:00&per_page=1", "before"},
	} {
		req := httptest.NewRequest(http.MethodGet, tc.path, nil)
		rec := httptest.NewRecorder()
		c = c.ForRequest(rec, req)
		(&rest.Server{}).Handle(rec, req, c)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("%s: status=%d body=%s", tc.path, rec.Code, rec.Body.String())
		}
		body := rec.Body.String()
		if !strings.Contains(body, "rest_invalid_param") || !strings.Contains(body, "rest_invalid_type") {
			t.Fatalf("%s: expected rest_invalid_type: %s", tc.path, body)
		}
		if !strings.Contains(body, tc.key+" is not of type string") {
			t.Fatalf("%s: expected string type error for %s: %s", tc.path, tc.key, body)
		}
	}
}

// P4.165: scalar after/before still filter inclusively (date_query inclusive=true).
func TestPostsDateQueryInclusiveScalarStillWorksP4165(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/posts?after=2020-01-01T00:00:00&before=2099-01-01T00:00:00&per_page=5", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, `"id"`) {
		t.Fatalf("expected posts in range: %s", body)
	}
}

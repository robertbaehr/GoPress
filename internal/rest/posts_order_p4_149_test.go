package rest_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/rest"
)

func TestPostsOrderEnumInvalidP4149(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	cases := []struct {
		path string
		key  string
		msg  string
	}{
		{"/wp-json/wp/v2/posts?order=foo&per_page=1", "order", "order is not one of asc and desc."},
		{"/wp-json/wp/v2/posts?order=&per_page=1", "order", "order is not one of asc and desc."},
		{"/wp-json/wp/v2/posts?orderby=nope&per_page=1", "orderby",
			"orderby is not one of author, date, id, include, modified, parent, relevance, slug, include_slugs, and title."},
		{"/wp-json/wp/v2/pages?orderby=nope&per_page=1", "orderby",
			"orderby is not one of author, date, id, include, modified, parent, relevance, slug, include_slugs, title, and menu_order."},
		{"/wp-json/wp/v2/posts?orderby=menu_order&per_page=1", "orderby",
			"orderby is not one of author, date, id, include, modified, parent, relevance, slug, include_slugs, and title."},
	}
	for _, tc := range cases {
		req := httptest.NewRequest(http.MethodGet, tc.path, nil)
		rec := httptest.NewRecorder()
		c = c.ForRequest(rec, req)
		(&rest.Server{}).Handle(rec, req, c)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("%s: status=%d body=%s", tc.path, rec.Code, rec.Body.String())
		}
		var body map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		if body["code"] != "rest_invalid_param" {
			t.Fatalf("%s: code=%v", tc.path, body["code"])
		}
		data, _ := body["data"].(map[string]any)
		params, _ := data["params"].(map[string]any)
		if params[tc.key] != tc.msg {
			t.Fatalf("%s: params[%s]=%v want %q", tc.path, tc.key, params[tc.key], tc.msg)
		}
	}
}

func TestPagesMenuOrderFilterP4149(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/pages?menu_order=99999&per_page=100&_fields=id", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var pages []map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &pages); err != nil {
		t.Fatal(err)
	}
	if len(pages) != 0 {
		t.Fatalf("expected empty for menu_order=99999, got %v", pages)
	}

	req = httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/pages?menu_order=abc&per_page=1", nil)
	rec = httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "menu_order is not of type integer") {
		t.Fatalf("expected integer type error: %s", rec.Body.String())
	}
}

func TestPostsOrderAscStillWorksP4149(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()
	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/posts?order=asc&orderby=id&per_page=2&_fields=id", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var posts []struct {
		ID int64 `json:"id"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &posts); err != nil {
		t.Fatal(err)
	}
	if len(posts) < 2 || posts[0].ID > posts[1].ID {
		t.Fatalf("expected id ASC: %+v", posts)
	}
}

package rest_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/robertbaehr/GoPress/internal/rest"
)

func TestPostsInvalidContext(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/posts?context=bogus", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["code"] != "rest_invalid_param" {
		t.Fatalf("code=%v", body["code"])
	}
	data, _ := body["data"].(map[string]any)
	params, _ := data["params"].(map[string]any)
	if params["context"] != "context is not one of view, embed, and edit." {
		t.Fatalf("params=%v", params)
	}
}

func TestPostsEmbedContextSubset(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/posts/1?context=embed", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"content", "status", "guid", "modified", "categories", "tags", "sticky", "format", "template", "meta", "class_list", "comment_status", "ping_status", "date_gmt", "modified_gmt"} {
		if _, ok := body[key]; ok {
			t.Fatalf("embed should omit %q: keys=%v", key, mapKeys(body))
		}
	}
	for _, key := range []string{"id", "date", "slug", "type", "link", "title", "excerpt", "author", "featured_media", "_links"} {
		if _, ok := body[key]; !ok {
			t.Fatalf("embed missing %q: keys=%v", key, mapKeys(body))
		}
	}
}

func TestPostsCollectionEditContextForbiddenAnon(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/posts?context=edit", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["code"] != "rest_forbidden_context" {
		t.Fatalf("code=%v", body["code"])
	}
	if body["message"] != "Sorry, you are not allowed to edit posts in this post type." {
		t.Fatalf("message=%v", body["message"])
	}
}

func mapKeys(m map[string]any) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

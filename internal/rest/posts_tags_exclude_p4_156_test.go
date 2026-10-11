package rest_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/robertbaehr/GoPress/internal/rest"
)

// P4.156: tags_exclude=notint → rest_invalid_param / rest_invalid_type.
func TestPostsTagsExcludeInvalidParam(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/posts?tags_exclude=notint", nil)
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
	if params["tags_exclude"] == nil {
		t.Fatalf("expected tags_exclude in params: %v", body)
	}
	details, _ := data["details"].(map[string]any)
	te, _ := details["tags_exclude"].(map[string]any)
	if te["code"] != "rest_invalid_type" {
		t.Fatalf("details.code=%v body=%v", te["code"], body)
	}
}

// P4.156: tags_exclude[]= empty array element fails integer validation.
func TestPostsTagsExcludeEmptyArrayElem(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/posts?tags_exclude[]=", nil)
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
		t.Fatalf("code=%v body=%s", body["code"], rec.Body.String())
	}
}

// P4.156: valid tags_exclude still filters (compat with P4.136).
func TestPostsTagsExcludeValidP4156(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/posts?tags_exclude=1&per_page=100", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

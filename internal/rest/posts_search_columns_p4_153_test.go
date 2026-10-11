package rest_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/rest"
)

// P4.153: search_columns=bogus → rest_invalid_param / rest_not_in_enum.
func TestPostsSearchColumnsInvalidEnumP4153(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/posts?search=Hello&search_columns=bogus&per_page=1", nil)
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
	wantMsg := "search_columns[0] is not one of post_title, post_content, and post_excerpt."
	data, _ := body["data"].(map[string]any)
	params, _ := data["params"].(map[string]any)
	if params["search_columns"] != wantMsg {
		t.Fatalf("params.search_columns=%v want %q", params["search_columns"], wantMsg)
	}
	details, _ := data["details"].(map[string]any)
	sc, _ := details["search_columns"].(map[string]any)
	if sc["code"] != "rest_not_in_enum" {
		t.Fatalf("details.code=%v body=%v", sc["code"], body)
	}
}

// P4.153: mixed valid+invalid indexes the invalid token after empties are skipped.
func TestPostsSearchColumnsMixedInvalidIndexP4153(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/posts?search=Hello&search_columns=post_title,,bogus&per_page=1", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "search_columns[1] is not one of") {
		t.Fatalf("expected index 1: %s", rec.Body.String())
	}
}

// P4.153: empty search_columns= remains valid (defaults to all columns).
func TestPostsSearchColumnsEmptyOKP4153(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/posts?search=Hello&search_columns=&per_page=1", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

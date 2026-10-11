package rest_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/rest"
)

// P4.154: format=bogus → rest_invalid_param / rest_not_in_enum.
func TestPostsFormatQueryInvalidP4154(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/posts?format=bogus&per_page=1", nil)
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
	wantMsg := "format[0] is not one of standard, aside, chat, gallery, link, image, quote, status, video, and audio."
	if params["format"] != wantMsg {
		t.Fatalf("params.format=%v want %q", params["format"], wantMsg)
	}
	details, _ := data["details"].(map[string]any)
	fmtDetail, _ := details["format"].(map[string]any)
	if fmtDetail["code"] != "rest_not_in_enum" {
		t.Fatalf("details.code=%v", fmtDetail["code"])
	}
}

// P4.154: format=aside,bogus indexes the invalid token; format[]=aside filters.
func TestPostsFormatQueryMixedAndArrayP4154(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/posts?format=aside,bogus&per_page=1", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("mixed status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "format[1] is not one of") {
		t.Fatalf("expected format[1] message: %s", rec.Body.String())
	}

	// Case-sensitive enum (WP).
	req = httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/posts?format=Aside&per_page=1", nil)
	rec = httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("Aside status=%d", rec.Code)
	}

	// format[]=status filters (fixture Post 15).
	req = httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/posts?format[]=status&per_page=10", nil)
	rec = httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("format[]=status status=%d body=%s", rec.Code, rec.Body.String())
	}
	var hits []rest.Post
	if err := json.Unmarshal(rec.Body.Bytes(), &hits); err != nil {
		t.Fatal(err)
	}
	if len(hits) == 0 {
		t.Fatal("expected hits for format[]=status")
	}
	for _, p := range hits {
		if p.Format != "status" {
			t.Fatalf("post %d format=%q", p.ID, p.Format)
		}
	}

	// format[]= empty → 400.
	req = httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/posts?format[]=&per_page=1", nil)
	rec = httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("format[]= status=%d body=%s", rec.Code, rec.Body.String())
	}

	// Empty scalar format= remains valid (no filter).
	req = httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/posts?format=&per_page=1", nil)
	rec = httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("format= status=%d", rec.Code)
	}
}

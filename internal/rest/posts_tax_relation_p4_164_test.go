package rest_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/robertbaehr/GoPress/internal/rest"
)

// P4.164: tax_relation=bogus → rest_invalid_param / rest_not_in_enum.
func TestPostsTaxRelationInvalidEnumP4164(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/posts?tax_relation=XOR&per_page=1", nil)
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
	details, _ := data["details"].(map[string]any)
	tr, _ := details["tax_relation"].(map[string]any)
	if tr["code"] != "rest_not_in_enum" {
		t.Fatalf("detail=%v", tr)
	}
}

// P4.164: tax_relation=OR with categories+tags still succeeds.
func TestPostsTaxRelationORValidP4164(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/posts?categories=1&tags=1&tax_relation=OR&per_page=5", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

// P4.164: categories[operator]=AND requires all listed terms.
func TestPostsCategoriesOperatorANDP4164(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	// Uncategorized (1) AND a likely-missing combo → zero or fewer hits than IN.
	reqIN := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/posts?categories=1,2&per_page=100&_fields=id", nil)
	recIN := httptest.NewRecorder()
	cIN := c.ForRequest(recIN, reqIN)
	(&rest.Server{}).Handle(recIN, reqIN, cIN)
	if recIN.Code != http.StatusOK {
		t.Fatalf("IN status=%d", recIN.Code)
	}
	var inPosts []map[string]any
	if err := json.Unmarshal(recIN.Body.Bytes(), &inPosts); err != nil {
		t.Fatal(err)
	}

	reqAND := httptest.NewRequest(http.MethodGet,
		"/wp-json/wp/v2/posts?categories[terms]=1,2&categories[operator]=AND&per_page=100&_fields=id", nil)
	recAND := httptest.NewRecorder()
	cAND := c.ForRequest(recAND, reqAND)
	(&rest.Server{}).Handle(recAND, reqAND, cAND)
	if recAND.Code != http.StatusOK {
		t.Fatalf("AND status=%d body=%s", recAND.Code, recAND.Body.String())
	}
	var andPosts []map[string]any
	if err := json.Unmarshal(recAND.Body.Bytes(), &andPosts); err != nil {
		t.Fatal(err)
	}
	if len(andPosts) > len(inPosts) {
		t.Fatalf("AND should not return more than IN: and=%d in=%d", len(andPosts), len(inPosts))
	}
}

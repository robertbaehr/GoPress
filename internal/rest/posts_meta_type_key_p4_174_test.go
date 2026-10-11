package rest_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/rest"
)

// P4.174: meta_type_key=CHAR rejected; BINARY accepted.
func TestPostsMetaTypeKeyEnumP4174(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/posts?meta_type_key=CHAR&per_page=1", nil)
	rec := httptest.NewRecorder()
	rc := c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, rc)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	data, _ := body["data"].(map[string]any)
	details, _ := data["details"].(map[string]any)
	mt, _ := details["meta_type_key"].(map[string]any)
	msg, _ := mt["message"].(string)
	if !strings.Contains(msg, "BINARY") {
		t.Fatalf("expected BINARY enum message, got %q", msg)
	}

	okReq := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/posts?meta_type_key=BINARY&meta_key=footnotes&meta_compare=EXISTS&per_page=1", nil)
	okRec := httptest.NewRecorder()
	okRC := c.ForRequest(okRec, okReq)
	(&rest.Server{}).Handle(okRec, okReq, okRC)
	if okRec.Code == http.StatusBadRequest && strings.Contains(okRec.Body.String(), "rest_not_in_enum") {
		t.Fatalf("BINARY should be accepted: %s", okRec.Body.String())
	}
}

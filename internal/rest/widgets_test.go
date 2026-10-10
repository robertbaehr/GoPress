package rest_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/robertbaehr/GoPress/internal/rest"
)

func TestWidgetsRESTGoneEmpty(t *testing.T) {
	c := bootOrSkip(t)
	t.Cleanup(func() { _ = c.DB.Close() })

	for _, path := range []string{
		"/wp-json/wp/v2/widgets",
		"/wp-json/wp/v2/widget-types",
		"/wp-json/wp/v2/sidebars",
	} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		rc := c.ForRequest(rec, req)
		(&rest.Server{}).Handle(rec, req, rc)
		if rec.Code != http.StatusOK {
			t.Fatalf("%s status=%d body=%s", path, rec.Code, rec.Body.String())
		}
		var out []any
		if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
			t.Fatalf("%s decode: %v body=%s", path, err, rec.Body.String())
		}
		if len(out) != 0 {
			t.Fatalf("%s expected empty, got %#v", path, out)
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/widgets/search-2", nil)
	rec := httptest.NewRecorder()
	rc := c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, rc)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("item status=%d body=%s", rec.Code, rec.Body.String())
	}
	var errBody map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &errBody); err != nil {
		t.Fatal(err)
	}
	if errBody["code"] != "rest_widgets_disabled" {
		t.Fatalf("code=%v", errBody["code"])
	}
}

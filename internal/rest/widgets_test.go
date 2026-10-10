package rest_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
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
		if rec.Header().Get("X-GoPress-Widgets") != "disabled" {
			t.Fatalf("%s missing disabled header", path)
		}
		var out []any
		if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
			t.Fatalf("%s decode: %v body=%s", path, err, rec.Body.String())
		}
		if len(out) != 0 {
			t.Fatalf("%s expected empty, got %#v", path, out)
		}
	}
}

func TestWidgetsRESTItemsDisabled(t *testing.T) {
	c := bootOrSkip(t)
	t.Cleanup(func() { _ = c.DB.Close() })

	for _, path := range []string{
		"/wp-json/wp/v2/widgets/search-2",
		"/wp-json/wp/v2/widget-types/search",
		"/wp-json/wp/v2/sidebars/sidebar-1",
	} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		rc := c.ForRequest(rec, req)
		(&rest.Server{}).Handle(rec, req, rc)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("%s status=%d body=%s", path, rec.Code, rec.Body.String())
		}
		var errBody map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &errBody); err != nil {
			t.Fatal(err)
		}
		if errBody["code"] != "rest_widgets_disabled" {
			t.Fatalf("%s code=%v", path, errBody["code"])
		}
	}
}

func TestWidgetsRESTWritesDisabled(t *testing.T) {
	c := bootOrSkip(t)
	t.Cleanup(func() { _ = c.DB.Close() })

	for _, tc := range []struct {
		method, path string
	}{
		{http.MethodPost, "/wp-json/wp/v2/sidebars"},
		{http.MethodPost, "/wp-json/wp/v2/widgets"},
		{http.MethodPut, "/wp-json/wp/v2/sidebars/sidebar-1"},
		{http.MethodDelete, "/wp-json/wp/v2/widgets/search-2"},
	} {
		req := httptest.NewRequest(tc.method, tc.path, strings.NewReader(`{}`))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		rc := c.ForRequest(rec, req)
		(&rest.Server{}).Handle(rec, req, rc)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("%s %s status=%d body=%s", tc.method, tc.path, rec.Code, rec.Body.String())
		}
		var errBody map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &errBody); err != nil {
			t.Fatal(err)
		}
		if errBody["code"] != "rest_widgets_disabled" {
			t.Fatalf("%s %s code=%v", tc.method, tc.path, errBody["code"])
		}
	}
}

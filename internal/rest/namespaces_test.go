package rest_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/robertbaehr/GoPress/internal/rest"
)

func TestNamespaceIndexWPV2(t *testing.T) {
	c := bootOrSkip(t)
	t.Cleanup(func() { _ = c.DB.Close() })
	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if rec.Header().Get("Allow") != "GET" {
		t.Fatalf("Allow=%q", rec.Header().Get("Allow"))
	}
	var body struct {
		Namespace string                    `json:"namespace"`
		Routes    map[string]map[string]any `json:"routes"`
		Links     map[string]any            `json:"_links"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("json: %v", err)
	}
	if body.Namespace != "wp/v2" {
		t.Fatalf("namespace=%q", body.Namespace)
	}
	if len(body.Routes) < 10 {
		t.Fatalf("expected many routes, got %d", len(body.Routes))
	}
	if _, ok := body.Routes["/wp/v2"]; !ok {
		t.Fatalf("missing /wp/v2 route entry")
	}
	if _, ok := body.Routes["/wp/v2/posts"]; !ok {
		t.Fatalf("missing /wp/v2/posts route entry")
	}
	if body.Links["up"] == nil {
		t.Fatalf("missing _links.up: %#v", body.Links)
	}
}

func TestNamespaceIndexHEAD(t *testing.T) {
	c := bootOrSkip(t)
	t.Cleanup(func() { _ = c.DB.Close() })
	req := httptest.NewRequest(http.MethodHead, "/wp-json/wp/v2", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d", rec.Code)
	}
	if rec.Body.Len() != 0 {
		t.Fatalf("HEAD should have empty body, got %q", rec.Body.String())
	}
}

func TestNamespaceIndexOEmbedAndSiteHealthAnon(t *testing.T) {
	c := bootOrSkip(t)
	t.Cleanup(func() { _ = c.DB.Close() })
	for _, ns := range []string{"oembed/1.0", "wp-site-health/v1", "wp-block-editor/v1", "wp-abilities/v1"} {
		req := httptest.NewRequest(http.MethodGet, "/wp-json/"+ns, nil)
		rec := httptest.NewRecorder()
		c2 := c.ForRequest(rec, req)
		(&rest.Server{}).Handle(rec, req, c2)
		if rec.Code != http.StatusOK {
			t.Fatalf("%s status=%d body=%s", ns, rec.Code, rec.Body.String())
		}
		var body map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatalf("%s json: %v", ns, err)
		}
		if body["namespace"] != ns {
			t.Fatalf("%s namespace=%v", ns, body["namespace"])
		}
		routes, _ := body["routes"].(map[string]any)
		if len(routes) == 0 {
			t.Fatalf("%s empty routes", ns)
		}
	}
}

func TestNamespaceIndexRestRouteQuery(t *testing.T) {
	c := bootOrSkip(t)
	t.Cleanup(func() { _ = c.DB.Close() })
	req := httptest.NewRequest(http.MethodGet, "/?rest_route=/wp/v2", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("json: %v", err)
	}
	if body["namespace"] != "wp/v2" {
		t.Fatalf("namespace=%v", body["namespace"])
	}
}

func TestNamespaceIndexUnknown(t *testing.T) {
	c := bootOrSkip(t)
	t.Cleanup(func() { _ = c.DB.Close() })
	req := httptest.NewRequest(http.MethodGet, "/wp-json/nope/v9", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

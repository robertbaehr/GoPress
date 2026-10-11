package rest_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/rest"
)

func TestOEmbedRequiresURL(t *testing.T) {
	c := bootOrSkip(t)
	t.Cleanup(func() { _ = c.DB.Close() })
	req := httptest.NewRequest(http.MethodGet, "/wp-json/oembed/1.0/embed", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestOEmbedPostByQueryID(t *testing.T) {
	c := bootOrSkip(t)
	t.Cleanup(func() { _ = c.DB.Close() })
	q := url.Values{}
	q.Set("url", "http://127.0.0.1:8081/?p=1")
	req := httptest.NewRequest(http.MethodGet, "/wp-json/oembed/1.0/embed?"+q.Encode(), nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var resp rest.OEmbedResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.Version != "1.0" || resp.Title == "" || resp.Type == "" {
		t.Fatalf("got %#v", resp)
	}
	if resp.ProviderName == "" {
		t.Fatal("missing provider_name")
	}
}

func TestOEmbedNamespaceAlias(t *testing.T) {
	// P4.127: GET /oembed/1.0 is the namespace index (embed lives at /oembed/1.0/embed).
	c := bootOrSkip(t)
	t.Cleanup(func() { _ = c.DB.Close() })
	q := url.Values{}
	q.Set("url", "http://127.0.0.1:8081/?p=1")
	req := httptest.NewRequest(http.MethodGet, "/wp-json/oembed/1.0?"+q.Encode(), nil)
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
	if body["namespace"] != "oembed/1.0" {
		t.Fatalf("expected namespace index, got %v", body["namespace"])
	}
	routes, _ := body["routes"].(map[string]any)
	if len(routes) == 0 {
		t.Fatalf("expected routes in namespace index: %v", body)
	}
}

func TestOEmbedInvalidURL(t *testing.T) {
	c := bootOrSkip(t)
	t.Cleanup(func() { _ = c.DB.Close() })
	q := url.Values{}
	q.Set("url", "http://127.0.0.1:8081/?p=999999999")
	req := httptest.NewRequest(http.MethodGet, "/wp-json/oembed/1.0/embed?"+q.Encode(), nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestOEmbedRichHTMLAndXML(t *testing.T) {
	// P3.99: iframe embed HTML + format=xml.
	c := bootOrSkip(t)
	t.Cleanup(func() { _ = c.DB.Close() })
	q := url.Values{}
	q.Set("url", "http://127.0.0.1:8081/?p=1")
	q.Set("maxwidth", "400")
	req := httptest.NewRequest(http.MethodGet, "/wp-json/oembed/1.0/embed?"+q.Encode(), nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var resp rest.OEmbedResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.Width != 400 || resp.Type != "rich" {
		t.Fatalf("got %#v", resp)
	}
	for _, want := range []string{
		`wp-embedded-content`,
		`data-secret=`,
		`<iframe`,
		`/embed/#?secret=`,
		`sandbox="allow-scripts"`,
	} {
		if !strings.Contains(resp.HTML, want) {
			t.Fatalf("missing %q in html %q", want, resp.HTML)
		}
	}

	q.Set("format", "xml")
	req = httptest.NewRequest(http.MethodGet, "/wp-json/oembed/1.0/embed?"+q.Encode(), nil)
	rec = httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("xml status=%d body=%s", rec.Code, rec.Body.String())
	}
	ct := rec.Header().Get("Content-Type")
	if !strings.Contains(ct, "xml") {
		t.Fatalf("content-type=%q", ct)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "<oembed>") || !strings.Contains(body, "<type>rich</type>") {
		t.Fatalf("xml body=%q", body)
	}
}

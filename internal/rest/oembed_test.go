package rest_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
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

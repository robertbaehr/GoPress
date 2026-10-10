package rest_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/rest"
)

func TestMediaCollection(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/media?per_page=3", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if rec.Header().Get("X-WP-Total") == "" {
		t.Fatal("expected X-WP-Total")
	}
	var items []rest.Media
	if err := json.Unmarshal(rec.Body.Bytes(), &items); err != nil {
		t.Fatalf("json: %v body=%s", err, rec.Body.String())
	}
	if len(items) == 0 || len(items) > 3 {
		t.Fatalf("len=%d", len(items))
	}
	m := items[0]
	if m.ID <= 0 || m.Type != "attachment" || m.Status != "inherit" {
		t.Fatalf("unexpected media: %+v", m)
	}
	if m.SourceURL == "" {
		t.Fatal("expected source_url")
	}
	if m.MimeType == "" || m.MediaType == "" {
		t.Fatalf("mime/media_type missing: %+v", m)
	}
	if m.GUID.Rendered == "" {
		t.Fatal("expected guid.rendered")
	}
}

func TestMediaSingle(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/media/5", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var m rest.Media
	if err := json.Unmarshal(rec.Body.Bytes(), &m); err != nil {
		t.Fatalf("json: %v", err)
	}
	if m.ID != 5 || m.Slug != "sample-photo-a" {
		t.Fatalf("unexpected: %+v", m)
	}
	if m.MediaType != "image" || !strings.HasPrefix(m.MimeType, "image/") {
		t.Fatalf("mime=%q media_type=%q", m.MimeType, m.MediaType)
	}
	if !strings.Contains(m.SourceURL, "sample-photo-a") {
		t.Fatalf("source_url=%q", m.SourceURL)
	}
}

func TestMediaMissing(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/media/999999991", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

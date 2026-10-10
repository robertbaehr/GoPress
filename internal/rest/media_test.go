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
	if m.Links == nil || m.Links["self"] == nil || m.Links["collection"] == nil {
		t.Fatalf("expected _links: %+v", m.Links)
	}
	if m.MediaDetails == nil {
		t.Fatal("expected media_details")
	}
	if m.MediaType != "image" && m.MediaType != "file" {
		t.Fatalf("media_type=%q want image|file", m.MediaType)
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
	details, ok := m.MediaDetails.(map[string]any)
	if !ok || details["width"] == nil || details["sizes"] == nil {
		t.Fatalf("media_details=%v", m.MediaDetails)
	}
	sizes, ok := details["sizes"].(map[string]any)
	if !ok || sizes["full"] == nil {
		t.Fatalf("sizes=%v", details["sizes"])
	}
	if m.Filename == "" || m.Links == nil {
		t.Fatalf("filename=%q links=%v", m.Filename, m.Links)
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

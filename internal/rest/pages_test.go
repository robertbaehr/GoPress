package rest_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/robertbaehr/GoPress/internal/rest"
)

func TestPagesListAnon(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/pages?per_page=5", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var pages []rest.Post
	if err := json.Unmarshal(rec.Body.Bytes(), &pages); err != nil {
		t.Fatalf("json array: %v body=%s", err, rec.Body.String())
	}
	if len(pages) == 0 || len(pages) > 5 {
		t.Fatalf("len=%d", len(pages))
	}
	p := pages[0]
	if p.ID <= 0 || p.Slug == "" || p.Status != "publish" || p.Type != "page" {
		t.Fatalf("unexpected page: %+v", p)
	}
	if p.Title.Rendered == "" {
		t.Fatal("expected title.rendered")
	}
	if p.Link == "" {
		t.Fatal("expected link")
	}
}

func TestPageSingle(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/pages/2", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var page rest.Post
	if err := json.Unmarshal(rec.Body.Bytes(), &page); err != nil {
		t.Fatalf("json: %v body=%s", err, rec.Body.String())
	}
	if page.ID != 2 || page.Type != "page" || page.Slug != "sample-page" {
		t.Fatalf("unexpected page: %+v", page)
	}
	if page.Title.Rendered == "" || page.Link == "" {
		t.Fatalf("missing title/link: %+v", page)
	}
}

func TestPageSingleMissing(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	req := httptest.NewRequest(http.MethodGet, "/wp-json/wp/v2/pages/999999991", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&rest.Server{}).Handle(rec, req, c)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

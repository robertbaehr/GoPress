package server_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/server"
)

func TestRobotsTXT(t *testing.T) {
	c := testBoot(t)
	srv := &server.Server{WP: c, AbsPath: "/workspace/", Front: &server.Front{}}
	req := httptest.NewRequest(http.MethodGet, "/robots.txt", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "User-agent: *") || !strings.Contains(body, "Disallow: /wp-admin/") {
		t.Fatalf("body=%q", body)
	}
	if !strings.Contains(body, "Sitemap:") || !strings.Contains(body, "/wp-sitemap.xml") {
		t.Fatalf("missing Sitemap line: %q", body)
	}
	ct := rec.Header().Get("Content-Type")
	if !strings.Contains(ct, "text/plain") {
		t.Fatalf("Content-Type=%q", ct)
	}
}

func TestSitemapIndexAndPosts(t *testing.T) {
	c := testBoot(t)
	srv := &server.Server{WP: c, AbsPath: "/workspace/", Front: &server.Front{}}

	req := httptest.NewRequest(http.MethodGet, "/wp-sitemap.xml", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("index status=%d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "<sitemapindex") || !strings.Contains(body, "wp-sitemap-posts-post-1.xml") {
		t.Fatalf("index body=%q", body)
	}

	req = httptest.NewRequest(http.MethodGet, "/wp-sitemap-posts-post-1.xml", nil)
	rec = httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("posts status=%d body=%s", rec.Code, rec.Body.String())
	}
	body = rec.Body.String()
	if !strings.Contains(body, "<urlset") || !strings.Contains(body, "<loc>") {
		t.Fatalf("posts body=%q", body)
	}
}

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

func TestRobotsTXTDisallowAllWhenNotPublic(t *testing.T) {
	c := testBoot(t)
	orig := c.Options.GetOption("blog_public", "1")
	t.Cleanup(func() { _, _ = c.Options.UpdateOption("blog_public", orig, "yes") })
	if _, err := c.Options.UpdateOption("blog_public", "0", "yes"); err != nil {
		t.Fatalf("UpdateOption: %v", err)
	}
	srv := &server.Server{WP: c, AbsPath: "/workspace/", Front: &server.Front{}}
	req := httptest.NewRequest(http.MethodGet, "/robots.txt", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Disallow: /\n") {
		t.Fatalf("expected Disallow: / got %q", body)
	}
	if strings.Contains(body, "Sitemap:") {
		t.Fatalf("unexpected Sitemap when private: %q", body)
	}
}

func TestRobotsTXTFilter(t *testing.T) {
	c := testBoot(t)
	c.Hooks.AddFilter("robots_txt", func(value any, args ...any) any {
		out, _ := value.(string)
		if len(args) < 1 {
			t.Fatal("expected public bool arg")
		}
		if _, ok := args[0].(bool); !ok {
			t.Fatalf("public arg type %T", args[0])
		}
		return out + "Disallow: /private/\n"
	}, 10, 2)
	srv := &server.Server{WP: c, AbsPath: "/workspace/", Front: &server.Front{}}
	req := httptest.NewRequest(http.MethodGet, "/robots.txt", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	body := rec.Body.String()
	if !strings.Contains(body, "Disallow: /private/") {
		t.Fatalf("expected robots_txt filter line: %q", body)
	}
	if !strings.Contains(body, "Sitemap:") {
		t.Fatalf("expected Sitemap after filter: %q", body)
	}
}

func TestSitemapDisabledWhenNotPublic(t *testing.T) {
	c := testBoot(t)
	orig := c.Options.GetOption("blog_public", "1")
	t.Cleanup(func() { _, _ = c.Options.UpdateOption("blog_public", orig, "yes") })
	if _, err := c.Options.UpdateOption("blog_public", "0", "yes"); err != nil {
		t.Fatalf("UpdateOption: %v", err)
	}
	srv := &server.Server{WP: c, AbsPath: "/workspace/", Front: &server.Front{}}
	req := httptest.NewRequest(http.MethodGet, "/wp-sitemap.xml", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status=%d want 404 body=%q", rec.Code, rec.Body.String())
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
	for _, want := range []string{
		"wp-sitemap-posts-page-1.xml",
		"wp-sitemap-taxonomies-category-1.xml",
		"wp-sitemap-taxonomies-post_tag-1.xml",
		"wp-sitemap-taxonomies-post_format-1.xml",
		"wp-sitemap-users-1.xml",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("index missing %s: %q", want, body)
		}
	}
	// Compact (non-pretty) markup like WP.
	if strings.Contains(body, "\t<sitemap>") {
		t.Fatalf("expected compact sitemap index, got pretty-printed: %q", body)
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

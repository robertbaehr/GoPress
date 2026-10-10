package server_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/config"
	"github.com/robertbaehr/GoPress/internal/server"
	"github.com/robertbaehr/GoPress/internal/wp"
)

func testBoot(t *testing.T) *wp.Context {
	t.Helper()
	cfg := &config.Config{
		DBName: "gopress_go", DBUser: "gopress", DBPassword: "gopress",
		DBHost: "127.0.0.1:3306", DBCharset: "utf8mb4", TablePrefix: "wp_",
		ABSPATH: "/workspace/",
		AuthKey: "k", SecureAuthKey: "k", LoggedInKey: "k", NonceKey: "k",
		AuthSalt: "s", SecureAuthSalt: "s", LoggedInSalt: "s", NonceSalt: "s",
	}
	c, err := wp.Boot(context.Background(), cfg)
	if err != nil {
		t.Skipf("Boot unavailable: %v", err)
	}
	t.Cleanup(func() { _ = c.DB.Close() })
	return c
}

func TestFrontHomeRendersBlogname(t *testing.T) {
	c := testBoot(t)
	srv := &server.Server{WP: c, AbsPath: "/workspace/", Front: &server.Front{}}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, "<title>") || !strings.Contains(body, "GoPress Fixture") {
		t.Fatalf("expected blogname in HTML, got %q", body)
	}
	// P5.5: active theme has templates/home.html — render blocks instead of shim.
	if !strings.Contains(body, `class="wp-template"`) {
		t.Fatalf("expected wp-template body, got %q", body)
	}
	if !strings.Contains(body, "wp-block-group") {
		t.Fatalf("expected rendered block template markup, got %q", body)
	}
	if strings.Contains(body, "<!-- wp:") {
		t.Fatalf("block comments should be rendered away, got %q", body)
	}
	if strings.Contains(body, `<ul class="posts">`) {
		t.Fatalf("shim home should not render when a block template exists")
	}
	if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "text/html") {
		t.Fatalf("content-type=%q", ct)
	}
}

func TestFrontSingularByQueryP(t *testing.T) {
	c := testBoot(t)
	srv := &server.Server{WP: c, AbsPath: "/workspace/", Front: &server.Front{}}
	req := httptest.NewRequest(http.MethodGet, "/?p=1", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Hello world!") {
		t.Fatalf("expected title, got %q", body)
	}
	if !strings.Contains(body, "entry-content") || !strings.Contains(body, "Welcome to WordPress") {
		t.Fatalf("expected content, got %q", body)
	}
	if strings.Contains(body, "<!-- wp:") {
		t.Fatalf("block comments should be rendered away, got %q", body)
	}
}

func TestFrontSingularByPageID(t *testing.T) {
	c := testBoot(t)
	srv := &server.Server{WP: c, AbsPath: "/workspace/", Front: &server.Front{}}
	req := httptest.NewRequest(http.MethodGet, "/?page_id=2", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "Sample Page") {
		t.Fatalf("expected Sample Page, got %q", rec.Body.String())
	}
}

func TestFrontSingularByPrettyPermalink(t *testing.T) {
	c := testBoot(t)
	srv := &server.Server{WP: c, AbsPath: "/workspace/", Front: &server.Front{}}
	req := httptest.NewRequest(http.MethodGet, "/2024/06/15/hello-world/", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "Hello world!") {
		t.Fatalf("expected Hello world!, got %q", rec.Body.String())
	}
}

func TestFrontSingular404(t *testing.T) {
	c := testBoot(t)
	srv := &server.Server{WP: c, AbsPath: "/workspace/", Front: &server.Front{}}
	req := httptest.NewRequest(http.MethodGet, "/?p=999999999", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "Page not found") {
		t.Fatalf("expected 404 page, got %q", rec.Body.String())
	}
}

func TestFrontAttachmentByQueryID(t *testing.T) {
	c := testBoot(t)
	srv := &server.Server{WP: c, AbsPath: "/workspace/", Front: &server.Front{}}
	req := httptest.NewRequest(http.MethodGet, "/?attachment_id=5", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, "sample-photo-a") {
		t.Fatalf("expected attachment title/url, got %q", body)
	}
	if !strings.Contains(body, "attachment") || !strings.Contains(body, "type-attachment") {
		t.Fatalf("expected attachment body/class, got %q", body)
	}
}

func TestFrontAttachmentByRewrite(t *testing.T) {
	c := testBoot(t)
	srv := &server.Server{WP: c, AbsPath: "/workspace/", Front: &server.Front{}}
	// Fixture rewrite_rules include ".?.+?/attachment/([^/]+)/?$" → attachment=$matches[1].
	req := httptest.NewRequest(http.MethodGet, "/files/attachment/sample-photo-a/", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "sample-photo-a") {
		t.Fatalf("expected attachment, got %q", rec.Body.String())
	}
}

func TestFrontFeedQueryRSS2(t *testing.T) {
	c := testBoot(t)
	srv := &server.Server{WP: c, AbsPath: "/workspace/", Front: &server.Front{}}
	req := httptest.NewRequest(http.MethodGet, "/?feed=rss2", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String())
	}
	ct := rec.Header().Get("Content-Type")
	if !strings.Contains(ct, "rss") && !strings.Contains(ct, "xml") {
		t.Fatalf("content-type=%q", ct)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `<rss version="2.0">`) || !strings.Contains(body, "<channel>") {
		t.Fatalf("expected RSS2 channel, got %q", body)
	}
	if !strings.Contains(body, "<item>") {
		t.Fatalf("expected feed items, got %q", body)
	}
}

func TestFrontFeedPrettyPath(t *testing.T) {
	c := testBoot(t)
	srv := &server.Server{WP: c, AbsPath: "/workspace/", Front: &server.Front{}}
	req := httptest.NewRequest(http.MethodGet, "/feed/", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `<rss version="2.0">`) {
		t.Fatalf("expected RSS2, got %q", rec.Body.String())
	}
}

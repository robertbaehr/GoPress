package wp_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestGetRawRefererFromQuery(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/wp-admin/options.php?_wp_http_referer=/wp-admin/options-general.php", nil)
	rec := httptest.NewRecorder()
	c := wp.NewContext(nil).ForRequest(rec, req)
	got := wp.GetRawReferer(c)
	if got != "/wp-admin/options-general.php" {
		t.Fatalf("query referer=%q", got)
	}
}

func TestGetRawRefererPrefersQueryOverHeader(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/x?_wp_http_referer=/from-query", nil)
	req.Header.Set("Referer", "http://evil.example/from-header")
	rec := httptest.NewRecorder()
	c := wp.NewContext(nil).ForRequest(rec, req)
	got := wp.GetRawReferer(c)
	if got != "/from-query" {
		t.Fatalf("prefer query, got %q", got)
	}
}

func TestGetRawRefererFromHeader(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/wp-admin/", nil)
	req.Header.Set("Referer", "http://127.0.0.1:8082/wp-admin/edit.php")
	rec := httptest.NewRecorder()
	c := wp.NewContext(nil).ForRequest(rec, req)
	got := wp.GetRawReferer(c)
	if got != "http://127.0.0.1:8082/wp-admin/edit.php" {
		t.Fatalf("header referer=%q", got)
	}
}

func TestGetRawRefererFromPostBody(t *testing.T) {
	form := url.Values{}
	form.Set("_wp_http_referer", "/wp-admin/plugins.php")
	req := httptest.NewRequest(http.MethodPost, "/wp-admin/options.php", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Referer", "http://example.test/ignored")
	rec := httptest.NewRecorder()
	c := wp.NewContext(nil).ForRequest(rec, req)
	got := wp.GetRawReferer(c)
	if got != "/wp-admin/plugins.php" {
		t.Fatalf("POST referer=%q", got)
	}
}

func TestGetRawRefererUnslashes(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/x?_wp_http_referer="+url.QueryEscape(`http://example.test/a\'b`), nil)
	rec := httptest.NewRecorder()
	c := wp.NewContext(nil).ForRequest(rec, req)
	got := wp.GetRawReferer(c)
	if strings.Contains(got, `\'`) {
		t.Fatalf("expected unslashed referer, got %q", got)
	}
}

func TestGetRawRefererEmpty(t *testing.T) {
	if wp.GetRawReferer(nil) != "" {
		t.Fatal("nil context")
	}
	req := httptest.NewRequest(http.MethodGet, "/wp-admin/", nil)
	rec := httptest.NewRecorder()
	c := wp.NewContext(nil).ForRequest(rec, req)
	if wp.GetRawReferer(c) != "" {
		t.Fatal("expected empty when no referer")
	}
}

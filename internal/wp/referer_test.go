package wp_test

import (
	"context"
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
	c := wp.NewContext(context.TODO()).ForRequest(rec, req)
	got := wp.GetRawReferer(c)
	if got != "/wp-admin/options-general.php" {
		t.Fatalf("query referer=%q", got)
	}
}

func TestGetRawRefererPrefersQueryOverHeader(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/x?_wp_http_referer=/from-query", nil)
	req.Header.Set("Referer", "http://evil.example/from-header")
	rec := httptest.NewRecorder()
	c := wp.NewContext(context.TODO()).ForRequest(rec, req)
	got := wp.GetRawReferer(c)
	if got != "/from-query" {
		t.Fatalf("prefer query, got %q", got)
	}
}

func TestGetRawRefererFromHeader(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/wp-admin/", nil)
	req.Header.Set("Referer", "http://127.0.0.1:8082/wp-admin/edit.php")
	rec := httptest.NewRecorder()
	c := wp.NewContext(context.TODO()).ForRequest(rec, req)
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
	c := wp.NewContext(context.TODO()).ForRequest(rec, req)
	got := wp.GetRawReferer(c)
	if got != "/wp-admin/plugins.php" {
		t.Fatalf("POST referer=%q", got)
	}
}

func TestGetRawRefererUnslashes(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/x?_wp_http_referer="+url.QueryEscape(`http://example.test/a\'b`), nil)
	rec := httptest.NewRecorder()
	c := wp.NewContext(context.TODO()).ForRequest(rec, req)
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
	c := wp.NewContext(context.TODO()).ForRequest(rec, req)
	if wp.GetRawReferer(c) != "" {
		t.Fatal("expected empty when no referer")
	}
}

func TestGetRefererValidatesLocal(t *testing.T) {
	c := bootWP(t)
	home := strings.TrimRight(wp.HomeURL(c, "", ""), "/")
	req := httptest.NewRequest(http.MethodGet, "/wp-admin/options.php", nil)
	req.Header.Set("Referer", home+"/wp-admin/edit.php")
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	got := wp.GetReferer(c)
	if got != home+"/wp-admin/edit.php" {
		t.Fatalf("GetReferer=%q want %q", got, home+"/wp-admin/edit.php")
	}
	// Relative same-origin path is accepted.
	req = httptest.NewRequest(http.MethodGet, "/wp-admin/options.php?_wp_http_referer=/wp-admin/plugins.php", nil)
	rec = httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	if got := wp.GetReferer(c); got != "/wp-admin/plugins.php" {
		t.Fatalf("relative GetReferer=%q", got)
	}
}

func TestGetRefererRejectsSelf(t *testing.T) {
	c := bootWP(t)
	req := httptest.NewRequest(http.MethodGet, "/wp-admin/options.php", nil)
	req.Header.Set("Referer", "/wp-admin/options.php")
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	if got := wp.GetReferer(c); got != "" {
		t.Fatalf("self referer should be empty, got %q", got)
	}
}

func TestGetRefererRejectsExternal(t *testing.T) {
	c := bootWP(t)
	req := httptest.NewRequest(http.MethodGet, "/wp-admin/", nil)
	req.Header.Set("Referer", "https://evil.example/phish")
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	if got := wp.GetReferer(c); got != "" {
		t.Fatalf("external referer should be empty, got %q", got)
	}
}

func TestValidateRedirectRelative(t *testing.T) {
	c := bootWP(t)
	if got := wp.ValidateRedirect(c, "/wp-admin/edit.php", ""); got != "/wp-admin/edit.php" {
		t.Fatalf("relative=%q", got)
	}
	if got := wp.ValidateRedirect(c, "https://evil.example/", "/fallback"); got != "/fallback" {
		t.Fatalf("evil=%q", got)
	}
}

func TestSanitizeRedirectStripsCRLF(t *testing.T) {
	got := wp.SanitizeRedirect("/wp-admin/%0d%0aSet-Cookie:x")
	if strings.Contains(strings.ToLower(got), "%0d") || strings.Contains(strings.ToLower(got), "%0a") {
		t.Fatalf("CRLF not stripped: %q", got)
	}
	if !strings.Contains(got, "/wp-admin/") {
		t.Fatalf("path lost: %q", got)
	}
	if got := wp.SanitizeRedirect("/a b"); got != "/a%20b" {
		t.Fatalf("space encode=%q", got)
	}
}

func TestValidateRedirectResolvesRelative(t *testing.T) {
	c := bootWP(t)
	req := httptest.NewRequest(http.MethodGet, "/wp-admin/options.php", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	got := wp.ValidateRedirect(c, "edit.php", "/fallback")
	if got != "/wp-admin/edit.php" {
		t.Fatalf("resolved relative=%q want /wp-admin/edit.php", got)
	}
}

func TestValidateRedirectRejectsBadSchemesAndNewlines(t *testing.T) {
	c := bootWP(t)
	if got := wp.ValidateRedirect(c, "javascript:alert(1)", "/fallback"); got != "/fallback" {
		t.Fatalf("javascript scheme=%q", got)
	}
	if got := wp.ValidateRedirect(c, "/wp-admin/\r\nLocation: https://evil.example/", "/fallback"); strings.Contains(got, "\n") || strings.Contains(got, "\r") {
		t.Fatalf("newline survived: %q", got)
	}
}

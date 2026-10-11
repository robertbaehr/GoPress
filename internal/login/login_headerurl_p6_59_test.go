package login_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/login"
	"github.com/robertbaehr/GoPress/internal/wp"
)

// P6.59: login_headerurl is passed through EscURL.
func TestLoginHeaderURLEscURLP659(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	c.Hooks.AddFilter("login_headerurl", func(value any, _ ...any) any {
		return "https://example.test/path with spaces"
	}, 10, 1)

	req := httptest.NewRequest(http.MethodGet, "/wp-login.php", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&login.Handler{}).Handle(rec, req, c)
	body := rec.Body.String()
	if !strings.Contains(body, `href="https://example.test/path%20with%20spaces"`) &&
		!strings.Contains(body, `href="https://example.test/path`) {
		t.Fatalf("expected EscURL'd login_headerurl: %q", body[:min(500, len(body))])
	}
	if strings.Contains(body, "path with spaces") {
		t.Fatalf("raw spaces should not remain in href: %q", body[:min(500, len(body))])
	}
}

// P6.59: deprecated login_headertitle seeds login_headertext.
func TestLoginHeaderTitleDeprecatedP659(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	wp.LastDeprecatedHookNotice = ""
	c.Hooks.AddFilter("login_headertitle", func(value any, _ ...any) any {
		return "Legacy Title Text"
	}, 10, 1)

	req := httptest.NewRequest(http.MethodGet, "/wp-login.php", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&login.Handler{}).Handle(rec, req, c)
	body := rec.Body.String()
	if !strings.Contains(body, ">Legacy Title Text</a>") {
		t.Fatalf("login_headertitle should seed text: %q", body[:min(500, len(body))])
	}
	if !strings.Contains(wp.LastDeprecatedHookNotice, "login_headertitle") {
		t.Fatalf("expected deprecated notice, got %q", wp.LastDeprecatedHookNotice)
	}
}

// P6.59: login_headertext still wins over deprecated title.
func TestLoginHeaderTextOverridesTitleP659(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	c.Hooks.AddFilter("login_headertitle", func(value any, _ ...any) any {
		return "Legacy"
	}, 10, 1)
	c.Hooks.AddFilter("login_headertext", func(value any, _ ...any) any {
		return "Modern Text"
	}, 10, 1)

	req := httptest.NewRequest(http.MethodGet, "/wp-login.php", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&login.Handler{}).Handle(rec, req, c)
	body := rec.Body.String()
	if !strings.Contains(body, ">Modern Text</a>") {
		t.Fatalf("login_headertext should win: %q", body[:min(500, len(body))])
	}
	if strings.Contains(body, ">Legacy</a>") {
		t.Fatalf("legacy title should be overridden")
	}
}

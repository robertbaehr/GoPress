package login_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/login"
	"github.com/robertbaehr/GoPress/internal/wp"
)

// P6.60: PHP empty("0") — headertitle "0" does not replace default text.
func TestLoginHeaderTitleEmptyZeroP660(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	wp.LastDeprecatedHookNotice = ""
	c.Hooks.AddFilter("login_headertitle", func(value any, _ ...any) any {
		return "0"
	}, 10, 1)

	req := httptest.NewRequest(http.MethodGet, "/wp-login.php", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&login.Handler{}).Handle(rec, req, c)
	body := rec.Body.String()
	if !strings.Contains(body, ">Powered by WordPress</a>") {
		t.Fatalf("empty('0') should keep default text: %q", body[:min(500, len(body))])
	}
	if strings.Contains(body, ">0</a>") {
		t.Fatalf("title '0' should not seed link text")
	}
	if !strings.Contains(wp.LastDeprecatedHookNotice, "login_headertitle") {
		t.Fatalf("expected deprecated notice, got %q", wp.LastDeprecatedHookNotice)
	}
}

// P6.60: whitespace-only headertitle seeds text (PHP empty(" ") is false).
func TestLoginHeaderTitleWhitespaceSeedsP660(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	c.Hooks.AddFilter("login_headertitle", func(value any, _ ...any) any {
		return "   "
	}, 10, 1)

	req := httptest.NewRequest(http.MethodGet, "/wp-login.php", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&login.Handler{}).Handle(rec, req, c)
	body := rec.Body.String()
	if !strings.Contains(body, "wp-login-logo") || !strings.Contains(body, "><a href=") {
		t.Fatalf("unexpected logo markup: %q", body[:min(400, len(body))])
	}
	// Link text is three spaces (unescaped), not the default label.
	if strings.Contains(body, ">Powered by WordPress</a>") {
		t.Fatalf("whitespace title should replace default text")
	}
}

// P6.60: login_headertext may contain markup (wp-login.php does not esc_html).
func TestLoginHeaderTextAllowsMarkupP660(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	c.Hooks.AddFilter("login_headertext", func(value any, _ ...any) any {
		return `<span class="site-name">Acme</span>`
	}, 10, 1)

	req := httptest.NewRequest(http.MethodGet, "/wp-login.php", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&login.Handler{}).Handle(rec, req, c)
	body := rec.Body.String()
	if !strings.Contains(body, `<span class="site-name">Acme</span>`) {
		t.Fatalf("expected raw markup in header text: %q", body[:min(500, len(body))])
	}
	if strings.Contains(body, `&lt;span`) {
		t.Fatalf("header text should not be HTML-escaped")
	}
}

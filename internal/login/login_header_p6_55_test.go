package login_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/login"
)

// P6.55: login_headerurl / login_headertext filter the logo; login_header action fires.
func TestLoginHeaderURLTextAndAction(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	actionFired := false
	c.Hooks.AddAction("login_header", func(_ ...any) {
		actionFired = true
	}, 10, 0)
	c.Hooks.AddFilter("login_headerurl", func(value any, _ ...any) any {
		return "https://example.test/home"
	}, 10, 1)
	c.Hooks.AddFilter("login_headertext", func(value any, _ ...any) any {
		return "Example Site"
	}, 10, 1)

	req := httptest.NewRequest(http.MethodGet, "/wp-login.php", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&login.Handler{}).Handle(rec, req, c)
	body := rec.Body.String()
	if !actionFired {
		t.Fatal("login_header action did not fire")
	}
	if !strings.Contains(body, `href="https://example.test/home"`) {
		t.Fatalf("login_headerurl not applied: %q", body[:min(400, len(body))])
	}
	if !strings.Contains(body, ">Example Site</a>") {
		t.Fatalf("login_headertext not applied")
	}
	if strings.Contains(body, "Powered by WordPress") {
		t.Fatalf("default logo text should be replaced")
	}
}

// P6.55: defaults remain wordpress.org / Powered by WordPress.
func TestLoginHeaderDefaults(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	req := httptest.NewRequest(http.MethodGet, "/wp-login.php", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&login.Handler{}).Handle(rec, req, c)
	body := rec.Body.String()
	want := `class="wp-login-logo"><a href="https://wordpress.org/">Powered by WordPress</a></h1>`
	if !strings.Contains(body, want) {
		t.Fatalf("default logo missing: %q", body[:min(500, len(body))])
	}
}

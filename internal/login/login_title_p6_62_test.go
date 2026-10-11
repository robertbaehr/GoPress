package login_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/login"
)

// P6.62: login_title filter can replace the <title> contents.
func TestLoginTitleFilterP662(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	c.Hooks.AddFilter("login_title", func(value any, args ...any) any {
		orig, _ := args[0].(string)
		if orig != "Log In" {
			t.Fatalf("second arg title=%q", orig)
		}
		return "Custom Login Title"
	}, 10, 2)

	req := httptest.NewRequest(http.MethodGet, "/wp-login.php", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&login.Handler{}).Handle(rec, req, c)
	body := rec.Body.String()
	if !strings.Contains(body, "<title>Custom Login Title</title>") {
		t.Fatalf("expected filtered title: %q", body[:min(500, len(body))])
	}
	if strings.Contains(body, "&#8212; WordPress</title>") {
		t.Fatalf("default title should be replaced")
	}
}

// P6.62: default title still uses title &lsaquo; blogname &#8212; WordPress.
func TestLoginTitleDefaultP662(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	orig, _ := c.Options.GetOption("blogname", "").(string)
	_, _ = c.Options.UpdateOption("blogname", "GoPress Fixture", "yes")
	t.Cleanup(func() { _, _ = c.Options.UpdateOption("blogname", orig, "yes") })

	req := httptest.NewRequest(http.MethodGet, "/wp-login.php", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&login.Handler{}).Handle(rec, req, c)
	body := rec.Body.String()
	if !strings.Contains(body, "<title>Log In &lsaquo; GoPress Fixture &#8212; WordPress</title>") {
		t.Fatalf("default title: %q", body[:min(500, len(body))])
	}
}

package wp_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestAuthCheckLoggedOut(t *testing.T) {
	c, err := wp.Boot(context.Background(), testBootConfig())
	if err != nil {
		t.Skip(err)
	}
	defer c.DB.Close()
	if c.AuthCheck() {
		t.Fatal("expected AuthCheck false when logged out")
	}
}

func TestAuthCheckLoggedIn(t *testing.T) {
	c, err := wp.Boot(context.Background(), testBootConfig())
	if err != nil {
		t.Skip(err)
	}
	defer c.DB.Close()
	user, err := c.GetUserBy("login", "admin")
	if err != nil || user == nil {
		t.Fatalf("admin: %v", err)
	}
	c.CurrentUser = user
	if !c.AuthCheck() {
		t.Fatal("expected AuthCheck true when logged in")
	}
}

func TestAuthCheckHTML(t *testing.T) {
	c, err := wp.Boot(context.Background(), testBootConfig())
	if err != nil {
		t.Skip(err)
	}
	defer c.DB.Close()
	html := c.AuthCheckHTML()
	if !strings.Contains(html, `id="wp-auth-check-wrap"`) {
		t.Fatalf("missing wrap: %q", html)
	}
	if !strings.Contains(html, `class="hidden"`) {
		t.Fatalf("expected same-domain hidden class: %q", html)
	}
	if !strings.Contains(html, "interim-login=1") {
		t.Fatalf("missing interim-login: %q", html)
	}
	if !strings.Contains(html, `id="wp-auth-check-form"`) {
		t.Fatalf("missing interim form: %q", html)
	}
	if !strings.Contains(html, "wp-auth-fallback") {
		t.Fatalf("missing fallback: %q", html)
	}
}

func TestAuthCheckHTMLCrossDomainFallback(t *testing.T) {
	c, err := wp.Boot(context.Background(), testBootConfig())
	if err != nil {
		t.Skip(err)
	}
	defer c.DB.Close()
	req := httptest.NewRequest(http.MethodGet, "http://other.example/wp-admin/", nil)
	req.Host = "other.example"
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	// Force home to a different host so login URL is cross-domain.
	_, _ = c.Options.UpdateOption("home", "http://login.example", "yes")
	_, _ = c.Options.UpdateOption("siteurl", "http://login.example", "yes")
	html := c.AuthCheckHTML()
	if !strings.Contains(html, `class="hidden fallback"`) {
		t.Fatalf("expected fallback wrap: %q", html)
	}
	if strings.Contains(html, `id="wp-auth-check-form"`) {
		t.Fatalf("cross-domain should omit interim iframe form: %q", html)
	}
	if !strings.Contains(html, "Please log in again") {
		t.Fatalf("expected fallback link: %q", html)
	}
}

func TestAuthCheckHTMLSameDomainFilter(t *testing.T) {
	c, err := wp.Boot(context.Background(), testBootConfig())
	if err != nil {
		t.Skip(err)
	}
	defer c.DB.Close()
	c.Hooks.AddFilter("wp_auth_check_same_domain", func(value any, args ...any) any {
		return false
	}, 10, 1)
	html := c.AuthCheckHTML()
	if !strings.Contains(html, `hidden fallback`) {
		t.Fatalf("filter should force fallback: %q", html)
	}
	if strings.Contains(html, `wp-auth-check-form`) {
		t.Fatalf("filtered cross-domain should omit form: %q", html)
	}
}

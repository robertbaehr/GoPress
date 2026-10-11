package login_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/login"
)

// P6.56: login_body_class filter can append custom body classes.
func TestLoginBodyClassFilterP656(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()
	c.Hooks.AddFilter("login_body_class", func(value any, args ...any) any {
		classes, _ := value.([]string)
		return append(classes, "custom-login-skin")
	}, 10, 2)

	req := httptest.NewRequest(http.MethodGet, "/wp-login.php", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&login.Handler{}).Handle(rec, req, c)
	body := rec.Body.String()
	for _, want := range []string{
		`login-action-login`,
		`wp-core-ui`,
		`admin-color-modern`,
		`locale-en-us`,
		`custom-login-skin`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing body class %q in: %q", want, body[:min(500, len(body))])
		}
	}
}

func TestLoginBodyClassInterimP656(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()
	req := httptest.NewRequest(http.MethodGet, "/wp-login.php?interim-login=1", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&login.Handler{}).Handle(rec, req, c)
	body := rec.Body.String()
	if !strings.Contains(body, "interim-login") {
		t.Fatalf("expected interim-login class: %q", body[:min(400, len(body))])
	}
	// Ensure login no-js prefix remains before filtered classes.
	if !strings.Contains(body, `body class="login no-js `) {
		t.Fatalf("expected login no-js prefix: %q", body[:min(400, len(body))])
	}
}

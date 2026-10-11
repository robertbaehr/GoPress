package login_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/login"
	"github.com/robertbaehr/GoPress/internal/wp"
)

// P6.54: login_site_html_link filters the backtoblog anchor on the login form.
func TestLoginSiteHTMLLinkFilter(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()
	c.Hooks.AddFilter("login_site_html_link", func(value any, _ ...any) any {
		return `<a href="/" class="custom-back">Custom site link</a>`
	}, 10, 1)

	req := httptest.NewRequest(http.MethodGet, "/wp-login.php", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&login.Handler{}).Handle(rec, req, c)
	body := rec.Body.String()
	if !strings.Contains(body, `id="backtoblog"`) {
		t.Fatalf("missing backtoblog: %q", body)
	}
	if !strings.Contains(body, `class="custom-back"`) || !strings.Contains(body, "Custom site link") {
		t.Fatalf("login_site_html_link not applied: %q", body)
	}
	if strings.Contains(body, "&larr; Go to") {
		t.Fatalf("default backtoblog text should be replaced: %q", body)
	}
}

// P6.54: resetpass form focuses pass1 and includes backtoblog.
func TestResetPassFooterFocusPass1(t *testing.T) {
	c := bootOrSkip(t)
	defer c.DB.Close()

	user, err := c.GetUserBy("login", "admin")
	if err != nil || user == nil {
		t.Fatalf("admin: %v", err)
	}
	key, err := wp.GetPasswordResetKey(c, user)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodGet, "/wp-login.php?action=rp&key="+key+"&login=admin", nil)
	rec := httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&login.Handler{}).Handle(rec, req, c)
	var rpCookie *http.Cookie
	for _, ck := range rec.Result().Cookies() {
		if strings.HasPrefix(ck.Name, "wp-resetpass-") {
			rpCookie = ck
		}
	}
	if rpCookie == nil {
		t.Fatal("expected resetpass cookie")
	}
	req = httptest.NewRequest(http.MethodGet, "/wp-login.php?action=rp", nil)
	req.AddCookie(rpCookie)
	rec = httptest.NewRecorder()
	c = c.ForRequest(rec, req)
	(&login.Handler{}).Handle(rec, req, c)
	body := rec.Body.String()
	if !strings.Contains(body, `id="backtoblog"`) {
		t.Fatalf("resetpass should include backtoblog: %q", body)
	}
	if !strings.Contains(body, `getElementById('pass1')`) {
		t.Fatalf("resetpass should focus pass1: %q", body)
	}
}

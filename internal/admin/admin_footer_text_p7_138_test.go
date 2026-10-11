package admin_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/admin"
	"github.com/robertbaehr/GoPress/internal/server"
	"github.com/robertbaehr/GoPress/internal/wp"
)

// P7.138: admin_footer_text / update_footer filters and in_admin_footer action.
func TestAdminFooterTextFiltersP7138(t *testing.T) {
	c := bootOrSkip(t)

	user, err := c.GetUserBy("login", "admin")
	if err != nil || user == nil {
		t.Fatalf("admin: %v", err)
	}
	set, err := c.SetAuthCookie(user, false, false, 0, "")
	if err != nil {
		t.Fatalf("cookie: %v", err)
	}

	var inFooter bool
	c.Hooks.AddAction("in_admin_footer", func(args ...any) {
		inFooter = true
	}, 10, 0)
	c.Hooks.AddFilter("admin_footer_text", func(value any, args ...any) any {
		return `<span id="footer-thankyou">Custom thank you</span>`
	}, 10, 1)
	c.Hooks.AddFilter("update_footer", func(value any, args ...any) any {
		return "Custom upgrade"
	}, 10, 1)

	srv := &server.Server{WP: c, AbsPath: "/workspace/", Front: &server.Front{}, Admin: &admin.Handler{}}
	req := httptest.NewRequest(http.MethodGet, "/wp-admin/", nil)
	req.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d", rec.Code)
	}
	body := rec.Body.String()
	if !inFooter {
		t.Fatal("expected in_admin_footer action")
	}
	if !strings.Contains(body, "Custom thank you") {
		t.Fatalf("missing admin_footer_text: %s", body[strings.Index(body, "wpfooter"):strings.Index(body, "wpfooter")+400])
	}
	if !strings.Contains(body, "Custom upgrade") {
		t.Fatal("missing update_footer")
	}
	if !strings.Contains(body, `id="footer-upgrade"`) || !strings.Contains(body, `id="footer-left"`) {
		t.Fatal("missing footer structure")
	}
}

// P7.138: default footer shows Version and thank-you markup.
func TestAdminFooterTextDefaultP7138(t *testing.T) {
	c := bootOrSkip(t)

	user, err := c.GetUserBy("login", "admin")
	if err != nil || user == nil {
		t.Fatalf("admin: %v", err)
	}
	set, err := c.SetAuthCookie(user, false, false, 0, "")
	if err != nil {
		t.Fatalf("cookie: %v", err)
	}
	srv := &server.Server{WP: c, AbsPath: "/workspace/", Front: &server.Front{}, Admin: &admin.Handler{}}
	req := httptest.NewRequest(http.MethodGet, "/wp-admin/", nil)
	req.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `id="footer-thankyou"`) {
		t.Fatal("missing footer-thankyou")
	}
	if !strings.Contains(body, "Version "+wp.Version) {
		t.Fatalf("missing Version %s", wp.Version)
	}
}

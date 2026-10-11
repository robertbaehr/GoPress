package admin_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/admin"
	"github.com/robertbaehr/GoPress/internal/server"
	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestWelcomePanelOnDashboardP7135(t *testing.T) {
	c := bootOrSkip(t)
	ck := loginCookie(t, c)
	user, _ := c.GetUserBy("login", "admin")
	if user != nil {
		_, _, _, _ = c.Meta.UpdateMetadata(wp.MetaUser, user.ID, "show_welcome_panel", 1, "")
	}
	srv := &server.Server{WP: c, AbsPath: "/workspace/", Front: &server.Front{}, Admin: &admin.Handler{}}
	req := httptest.NewRequest(http.MethodGet, "/wp-admin/", nil)
	req.AddCookie(ck)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	body := rec.Body.String()
	for _, want := range []string{
		`id="welcome-panel"`,
		`Welcome to WordPress!`,
		`welcomepanelnonce`,
		`welcome-panel-close`,
		`Open site editor`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing %q in dashboard (len=%d)", want, len(body))
		}
	}
}

func TestWelcomePanelDismissP7135(t *testing.T) {
	c := bootOrSkip(t)
	ck := loginCookie(t, c)
	user, err := c.GetUserBy("login", "admin")
	if err != nil || user == nil {
		t.Fatalf("admin: %v", err)
	}
	_, _, _, _ = c.Meta.UpdateMetadata(wp.MetaUser, user.ID, "show_welcome_panel", 1, "")
	srv := &server.Server{WP: c, AbsPath: "/workspace/", Front: &server.Front{}, Admin: &admin.Handler{}}
	req := httptest.NewRequest(http.MethodGet, "/wp-admin/?welcome=0", nil)
	req.AddCookie(ck)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusFound {
		t.Fatalf("status=%d want 302 body=%s", rec.Code, rec.Body.String()[:min(200, rec.Body.Len())])
	}
	raw := c.Meta.GetMetadata(wp.MetaUser, user.ID, "show_welcome_panel", true)
	if strings.TrimSpace(fmt.Sprint(raw)) != "0" {
		t.Fatalf("show_welcome_panel=%v want 0", raw)
	}
}

func TestAjaxUpdateWelcomePanelP7135(t *testing.T) {
	c := bootOrSkip(t)
	ck := loginCookie(t, c)
	user, err := c.GetUserBy("login", "admin")
	if err != nil || user == nil {
		t.Fatalf("admin: %v", err)
	}
	_, _, _, _ = c.Meta.UpdateMetadata(wp.MetaUser, user.ID, "show_welcome_panel", 1, "")
	srv := &server.Server{WP: c, AbsPath: "/workspace/", Front: &server.Front{}, Admin: &admin.Handler{}}

	req := httptest.NewRequest(http.MethodGet, "/wp-admin/", nil)
	req.AddCookie(ck)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	m := regexp.MustCompile(`id="welcomepanelnonce"[^>]*value="([^"]+)"`).FindStringSubmatch(rec.Body.String())
	if len(m) < 2 {
		t.Fatalf("welcomepanelnonce not found")
	}
	nonce := m[1]

	form := url.Values{}
	form.Set("action", "update-welcome-panel")
	form.Set("welcomepanelnonce", nonce)
	form.Set("visible", "0")
	req = httptest.NewRequest(http.MethodPost, "/wp-admin/admin-ajax.php", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(ck)
	rec = httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || rec.Body.String() != "1" {
		t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String())
	}
	raw := c.Meta.GetMetadata(wp.MetaUser, user.ID, "show_welcome_panel", true)
	if strings.TrimSpace(fmt.Sprint(raw)) != "0" {
		t.Fatalf("show_welcome_panel=%v want 0", raw)
	}
}

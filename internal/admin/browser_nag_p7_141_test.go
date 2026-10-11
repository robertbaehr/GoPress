package admin_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/admin"
	"github.com/robertbaehr/GoPress/internal/server"
)

// P7.141: default CheckBrowserVersion is nil (no browse-happy API).
func TestBrowserNagAbsentByDefaultP7141(t *testing.T) {
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
	if strings.Contains(rec.Body.String(), "dashboard_browser_nag") {
		t.Fatal("expected no browser nag when CheckBrowserVersion is nil")
	}
}

// P7.141: injected insecure check emits dashboard_browser_nag widget.
func TestBrowserNagInjectedP7141(t *testing.T) {
	c := bootOrSkip(t)

	user, err := c.GetUserBy("login", "admin")
	if err != nil || user == nil {
		t.Fatalf("admin: %v", err)
	}
	set, err := c.SetAuthCookie(user, false, false, 0, "")
	if err != nil {
		t.Fatalf("cookie: %v", err)
	}

	c.Hooks.AddFilter("pre_wp_check_browser_version", func(value any, args ...any) any {
		return &admin.BrowserVersionCheck{
			Name:      "Chrome",
			Upgrade:   true,
			Insecure:  true,
			UpdateURL: "https://www.google.com/chrome/",
			ImgSrc:    "https://example.test/chrome.png",
		}
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
	for _, want := range []string{
		`dashboard_browser_nag`,
		`browser-insecure`,
		`You are using an insecure browser!`,
		`Chrome`,
		`browse happy`,
		`has-browser-icon`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing %q in dashboard body", want)
		}
	}
}

func TestCheckBrowserVersionUnitP7141(t *testing.T) {
	c := bootOrSkip(t)
	if got := admin.CheckBrowserVersion(c); got != nil {
		t.Fatalf("default check=%+v", got)
	}
	c.Hooks.AddFilter("pre_wp_check_browser_version", func(value any, args ...any) any {
		return &admin.BrowserVersionCheck{Name: "Firefox", Upgrade: true}
	}, 10, 1)
	got := admin.CheckBrowserVersion(c)
	if got == nil || got.Name != "Firefox" {
		t.Fatalf("filtered check=%+v", got)
	}
}

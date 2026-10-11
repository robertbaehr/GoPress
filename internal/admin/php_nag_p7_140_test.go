package admin_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/admin"
	"github.com/robertbaehr/GoPress/internal/server"
)

// P7.140: default CheckPHPVersion is nil (no PHP runtime).
func TestPHPNagAbsentByDefaultP7140(t *testing.T) {
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
	if strings.Contains(rec.Body.String(), "dashboard_php_nag") {
		t.Fatal("expected no php nag when CheckPHPVersion is nil")
	}
}

// P7.140: injected unacceptable check emits dashboard_php_nag widget.
func TestPHPNagInjectedP7140(t *testing.T) {
	c := bootOrSkip(t)

	user, err := c.GetUserBy("login", "admin")
	if err != nil || user == nil {
		t.Fatalf("admin: %v", err)
	}
	set, err := c.SetAuthCookie(user, false, false, 0, "")
	if err != nil {
		t.Fatalf("cookie: %v", err)
	}

	c.Hooks.AddFilter("pre_wp_check_php_version", func(value any, args ...any) any {
		return &admin.PHPVersionCheck{
			IsAcceptable:             false,
			IsSecure:                 false,
			IsLowerThanFutureMinimum: true,
			RecommendedVersion:       "8.3",
			Version:                  "7.4.33",
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
		`dashboard_php_nag`,
		`php-no-security-updates`,
		`PHP Update Required`,
		`7.4.33`,
		`Learn more about updating PHP`,
		`8.3`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing %q in dashboard body", want)
		}
	}
}

func TestCheckPHPVersionUnitP7140(t *testing.T) {
	c := bootOrSkip(t)
	if got := admin.CheckPHPVersion(c); got != nil {
		t.Fatalf("default check=%+v", got)
	}
	c.Hooks.AddFilter("pre_wp_check_php_version", func(value any, args ...any) any {
		return &admin.PHPVersionCheck{IsAcceptable: false, Version: "7.2"}
	}, 10, 1)
	got := admin.CheckPHPVersion(c)
	if got == nil || got.Version != "7.2" {
		t.Fatalf("filtered check=%+v", got)
	}
	if url := admin.UpdatePHPURL(c); !strings.Contains(url, "update-php") {
		t.Fatalf("UpdatePHPURL=%q", url)
	}
}

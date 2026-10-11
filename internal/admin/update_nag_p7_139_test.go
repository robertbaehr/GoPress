package admin_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/admin"
	"github.com/robertbaehr/GoPress/internal/server"
)

// P7.139: default PreferredCoreUpdate is nil (core updates out of scope).
func TestUpdateNagAbsentByDefaultP7139(t *testing.T) {
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
	if strings.Contains(rec.Body.String(), "update-nag") {
		t.Fatal("expected no update-nag when core updates are out of scope")
	}
}

// P7.139: injected upgrade response emits update-nag notice; skipped on update-core.php.
func TestUpdateNagInjectedAndSkippedP7139(t *testing.T) {
	c := bootOrSkip(t)

	user, err := c.GetUserBy("login", "admin")
	if err != nil || user == nil {
		t.Fatalf("admin: %v", err)
	}
	set, err := c.SetAuthCookie(user, false, false, 0, "")
	if err != nil {
		t.Fatalf("cookie: %v", err)
	}

	c.Hooks.AddFilter("pre_site_transient_update_core", func(value any, args ...any) any {
		return &admin.CoreUpdate{Response: "upgrade", Current: "9.9.9"}
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
	idx := strings.Index(body, "update-nag")
	snippet := body
	if idx >= 0 {
		end := idx + 400
		if end > len(body) {
			end = len(body)
		}
		snippet = body[idx:end]
	}
	for _, want := range []string{
		`update-nag`,
		`notice-warning`,
		`WordPress 9.9.9`,
		`version-9-9-9`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing %q in dashboard body; nag snippet=%q", want, snippet)
		}
	}
	if !strings.Contains(body, "Please update now") && !strings.Contains(body, "Please notify the site administrator") {
		t.Fatalf("missing update CTA; nag snippet=%q", snippet)
	}

	req = httptest.NewRequest(http.MethodGet, "/wp-admin/update-core.php", nil)
	req.AddCookie(&http.Cookie{Name: set.LoggedInName, Value: set.LoggedInValue})
	rec = httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("update-core status=%d", rec.Code)
	}
	if strings.Contains(rec.Body.String(), "update-nag") {
		t.Fatal("update-nag must be skipped on update-core.php")
	}
}

func TestUpdateNagUnitP7139(t *testing.T) {
	c := bootOrSkip(t)
	if got := admin.PreferredCoreUpdate(c); got != nil {
		t.Fatalf("default preferred=%+v", got)
	}
	c.Hooks.AddFilter("pre_site_transient_update_core", func(value any, args ...any) any {
		return &admin.CoreUpdate{Response: "upgrade", Current: "8.0"}
	}, 10, 1)
	got := admin.PreferredCoreUpdate(c)
	if got == nil || got.Current != "8.0" {
		t.Fatalf("filtered preferred=%+v", got)
	}

	req := httptest.NewRequest(http.MethodGet, "/wp-admin/index.php", nil)
	rec := httptest.NewRecorder()
	rc := c.ForRequest(rec, req)
	user, _ := c.GetUserBy("login", "admin")
	rc.CurrentUser = user
	var buf strings.Builder
	rc.Hooks.BeginNoticeCapture(&buf)
	admin.UpdateNag(rc)
	rc.Hooks.EndNoticeCapture()
	out := buf.String()
	if !strings.Contains(out, "update-nag") || !strings.Contains(out, "WordPress 8.0") {
		t.Fatalf("nag markup=%q", out)
	}
}

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

func TestDashboardGlanceVersionTheme(t *testing.T) {
	c := bootOrSkip(t)
	t.Cleanup(func() { _ = c.DB.Close() })

	c.Hooks.AddAction("rightnow_end", func(args ...any) {
		c.Hooks.WriteNotice(`<p class="rightnow-extra">Extra glance</p>`)
	}, 10, 0)

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
	for _, want := range []string{
		`id="wp-version"`,
		"WordPress " + wp.Version,
		"running",
		`href="themes.php"`,
		"Twenty Twenty-Five",
		`class="sub"`,
		"rightnow-extra",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing %q in dashboard (len=%d)", want, len(body))
		}
	}
	if c.Hooks.DidAction("rightnow_end") < 1 {
		t.Fatal("expected rightnow_end")
	}
	if c.Hooks.DidAction("activity_box_end") < 1 {
		t.Fatal("expected activity_box_end")
	}
}

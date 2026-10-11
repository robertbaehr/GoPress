package admin_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/admin"
	"github.com/robertbaehr/GoPress/internal/server"
)

func TestDashboardRightNowDeepen(t *testing.T) {
	// P7.110: At a Glance publish links, moderation count, glance filter, recent posts.
	c := bootOrSkip(t)
	t.Cleanup(func() { _ = c.DB.Close() })

	c.Hooks.AddFilter("dashboard_glance_items", func(value any, args ...any) any {
		return []string{`<a href="/wp-admin/">Glance Extra</a>`}
	}, 10, 1)

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
		"At a Glance",
		"post_status=publish",
		"Published post",
		"comment-mod-count",
		"comment_status=moderated",
		"Glance Extra",
		"published-posts",
		"Recently Published",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing %q in dashboard body (len=%d)", want, len(body))
		}
	}
}

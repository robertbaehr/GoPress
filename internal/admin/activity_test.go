package admin_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/admin"
	"github.com/robertbaehr/GoPress/internal/server"
)

func TestDashboardActivityRecentComments(t *testing.T) {
	c := bootOrSkip(t)
	ck := loginCookie(t, c)
	srv := &server.Server{WP: c, AbsPath: "/workspace/", Front: &server.Front{}, Admin: &admin.Handler{}}

	req := httptest.NewRequest(http.MethodGet, "/wp-admin/", nil)
	req.AddCookie(ck)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `id="latest-comments"`) {
		t.Fatalf("missing latest-comments: %q", body)
	}
	if !strings.Contains(body, `id="the-comment-list"`) {
		t.Fatalf("expected comment list: %q", body)
	}
	// Fixture includes "A WordPress Commenter" / "Hi, this is a comment."
	if !strings.Contains(body, "A WordPress Commenter") && !strings.Contains(body, "Commenter") {
		t.Fatalf("expected recent comment author from fixture: %q", body)
	}
	if strings.Contains(body, "No comments yet.") {
		t.Fatalf("should not show empty stub when comments exist: %q", body)
	}
}

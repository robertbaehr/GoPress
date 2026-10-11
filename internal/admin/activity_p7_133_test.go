package admin_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/admin"
	"github.com/robertbaehr/GoPress/internal/server"
)

func TestDashboardActivityRowActions(t *testing.T) {
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
	for _, want := range []string{
		`class="comment comment-item approved"`,
		`has-row-actions`,
		`cite class="comment-author"`,
		`Unapprove`,
		`action=unapprovecomment`,
		`_wpnonce=`,
		`action=spamcomment`,
		`action=trashcomment`,
		`vim-r button-link`,
		`aria-label="Edit this comment"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing %q in dashboard activity", want)
		}
	}
}

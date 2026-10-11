package admin_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/admin"
	"github.com/robertbaehr/GoPress/internal/server"
)

func TestEditPostsHelpTabs(t *testing.T) {
	c := bootOrSkip(t)
	ck := loginCookie(t, c)
	srv := &server.Server{WP: c, AbsPath: "/workspace/", Admin: &admin.Handler{}}

	req := httptest.NewRequest(http.MethodGet, "/wp-admin/edit.php", nil)
	req.AddCookie(ck)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d", rec.Code)
	}
	body := rec.Body.String()
	for _, want := range []string{
		`id="contextual-help-wrap"`,
		`id="contextual-help-link"`,
		`aria-controls="contextual-help-wrap"`,
		`id="tab-link-overview"`,
		`id="tab-panel-overview"`,
		`id="tab-link-screen-content"`,
		`id="tab-link-action-links"`,
		`id="tab-link-bulk-actions"`,
		`contextual-help-sidebar`,
		"Documentation on Managing Posts",
		">Help</button>",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected help chrome %q in edit.php", want)
		}
	}
}

func TestEditPagesHelpTabs(t *testing.T) {
	c := bootOrSkip(t)
	ck := loginCookie(t, c)
	srv := &server.Server{WP: c, AbsPath: "/workspace/", Admin: &admin.Handler{}}

	req := httptest.NewRequest(http.MethodGet, "/wp-admin/edit.php?post_type=page", nil)
	req.AddCookie(ck)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d", rec.Code)
	}
	body := rec.Body.String()
	for _, want := range []string{
		`id="tab-link-overview"`,
		`id="tab-link-managing-pages"`,
		"Documentation on Managing Pages",
		`id="contextual-help-link"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected page help chrome %q", want)
		}
	}
}

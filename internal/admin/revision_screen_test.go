package admin_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/admin"
	"github.com/robertbaehr/GoPress/internal/server"
)

func TestRevisionCompareScreenDeepen(t *testing.T) {
	c := bootOrSkip(t)
	ck := loginCookie(t, c)
	srv := &server.Server{WP: c, AbsPath: "/workspace/", Front: &server.Front{}, Admin: &admin.Handler{}}

	// Fixture revision 53 belongs to post 13 (Fixture Post 01).
	req := httptest.NewRequest(http.MethodGet, "/wp-admin/revision.php?revision=53", nil)
	req.AddCookie(ck)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	for _, want := range []string{
		`revisions-php`,
		`Compare Revisions of`,
		`Fixture Post 01`,
		`Return to editor`,
		`post.php?post=13`,
		`class="diff"`,
		`<th colspan="3">Title</th>`,
		`<th colspan="3">Content</th>`,
		`revision=53`,
		`revision_title=`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing %q in body", want)
		}
	}
}

func TestRevisionCompareScreenStubWithoutRevision(t *testing.T) {
	c := bootOrSkip(t)
	ck := loginCookie(t, c)
	srv := &server.Server{WP: c, AbsPath: "/workspace/", Front: &server.Front{}, Admin: &admin.Handler{}}

	req := httptest.NewRequest(http.MethodGet, "/wp-admin/revision.php", nil)
	req.AddCookie(ck)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Compare Revisions") {
		t.Fatalf("expected heading: %q", body)
	}
	if !strings.Contains(body, "select a revision") {
		t.Fatalf("expected stub description: %q", body)
	}
}

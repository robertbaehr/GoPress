package admin_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/admin"
	"github.com/robertbaehr/GoPress/internal/server"
)

func TestPostNewCreatesAutoDraftShell(t *testing.T) {
	c := bootOrSkip(t)
	ck := loginCookie(t, c)
	srv := &server.Server{WP: c, AbsPath: "/workspace/", Front: &server.Front{}, Admin: &admin.Handler{}}

	req := httptest.NewRequest(http.MethodGet, "/wp-admin/post-new.php", nil)
	req.AddCookie(ck)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	for _, want := range []string{
		`post-new-php`,
		`Add New Post`,
		`id="title"`,
		`id="content"`,
		`value="Publish"`,
		`name="post_ID"`,
		`value="auto-draft"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing %q in %q", want, body)
		}
	}
	// Title input should be empty for new posts (WP auto-draft UX).
	if strings.Contains(body, `id="title" spellcheck="true" autocomplete="off" />`) {
		// ok empty value=
	} else if !strings.Contains(body, `value="" id="title"`) && !strings.Contains(body, `name="post_title" size="30" value=""`) {
		t.Fatalf("expected empty title value: %q", body)
	}
}

func TestPostNewPageType(t *testing.T) {
	c := bootOrSkip(t)
	ck := loginCookie(t, c)
	srv := &server.Server{WP: c, AbsPath: "/workspace/", Front: &server.Front{}, Admin: &admin.Handler{}}

	req := httptest.NewRequest(http.MethodGet, "/wp-admin/post-new.php?post_type=page", nil)
	req.AddCookie(ck)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "Add New Page") {
		t.Fatalf("expected Add New Page: %q", rec.Body.String())
	}
}

func TestPostNewAttachmentRedirects(t *testing.T) {
	c := bootOrSkip(t)
	ck := loginCookie(t, c)
	srv := &server.Server{WP: c, AbsPath: "/workspace/", Front: &server.Front{}, Admin: &admin.Handler{}}

	req := httptest.NewRequest(http.MethodGet, "/wp-admin/post-new.php?post_type=attachment", nil)
	req.AddCookie(ck)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusFound {
		t.Fatalf("status=%d", rec.Code)
	}
	if loc := rec.Header().Get("Location"); loc != "/wp-admin/media-new.php" {
		t.Fatalf("Location=%q", loc)
	}
}

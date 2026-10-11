package admin_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/admin"
	"github.com/robertbaehr/GoPress/internal/server"
	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestEditCommentsPerPageScreenOptions(t *testing.T) {
	c := bootOrSkip(t)
	ck := loginCookie(t, c)
	srv := &server.Server{WP: c, AbsPath: "/workspace/", Admin: &admin.Handler{}}

	req := httptest.NewRequest(http.MethodGet, "/wp-admin/edit-comments.php", nil)
	req.AddCookie(ck)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d", rec.Code)
	}
	body := rec.Body.String()
	for _, want := range []string{
		`id="screen-options-wrap"`,
		`name="wp_screen_options[value]"`,
		`name="wp_screen_options[option]"`,
		`value="edit_comments_per_page"`,
		`id="edit_comments_per_page"`,
		`id="screenoptionnonce"`,
		"Number of items per page",
		"Screen Options",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected screen options %q in edit-comments.php", want)
		}
	}

	recN := httptest.NewRecorder()
	reqN := httptest.NewRequest(http.MethodGet, "/wp-admin/edit-comments.php", nil)
	reqN.AddCookie(ck)
	pc := c.ForRequest(recN, reqN)
	pc.AuthenticateRequest()
	if pc.CurrentUser == nil {
		t.Fatal("expected authenticated user")
	}
	uid := pc.CurrentUser.ID
	prev := wp.GetUserOption(pc, "edit_comments_per_page", uid)
	t.Cleanup(func() {
		if prev == "" {
			_, _ = pc.Meta.DeleteMetadata(wp.MetaUser, uid, pc.DB.TablePrefix+"edit_comments_per_page", "")
		} else {
			_ = wp.UpdateUserOption(pc, uid, "edit_comments_per_page", prev)
		}
	})

	form := url.Values{}
	form.Set("wp_screen_options[option]", "edit_comments_per_page")
	form.Set("wp_screen_options[value]", "2")
	form.Set("screenoptionnonce", pc.CreateNonce("screen-options-nonce"))
	form.Set("screen-options-apply", "Apply")
	req = httptest.NewRequest(http.MethodPost, "/wp-admin/edit-comments.php", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(ck)
	rec = httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusFound {
		t.Fatalf("screen options POST status=%d body=%q", rec.Code, rec.Body.String())
	}

	recR := httptest.NewRecorder()
	reqR := httptest.NewRequest(http.MethodGet, "/wp-admin/edit-comments.php", nil)
	reqR.AddCookie(ck)
	rc := c.ForRequest(recR, reqR)
	rc.AuthenticateRequest()
	if got := wp.GetUserOption(rc, "edit_comments_per_page", uid); got != "2" {
		t.Fatalf("saved per_page=%q want 2", got)
	}

	req = httptest.NewRequest(http.MethodGet, "/wp-admin/edit-comments.php", nil)
	req.AddCookie(ck)
	rec = httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("reload status=%d", rec.Code)
	}
	body = rec.Body.String()
	if !strings.Contains(body, `id="edit_comments_per_page"`) || !strings.Contains(body, `value="2"`) {
		t.Fatalf("expected per_page=2 in screen options")
	}
}

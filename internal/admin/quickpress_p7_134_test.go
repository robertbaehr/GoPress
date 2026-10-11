package admin_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/admin"
	"github.com/robertbaehr/GoPress/internal/server"
)

func TestQuickDraftFormHasNonceAndRecentDraftsP7134(t *testing.T) {
	c := bootOrSkip(t)
	ck := loginCookie(t, c)
	srv := &server.Server{WP: c, AbsPath: "/workspace/", Front: &server.Front{}, Admin: &admin.Handler{}}
	req := httptest.NewRequest(http.MethodGet, "/wp-admin/", nil)
	req.AddCookie(ck)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	body := rec.Body.String()
	for _, needle := range []string{
		`id="quick-press"`,
		`class="hide-if-no-js"`,
		`name="_wpnonce"`,
		`name="post_ID"`,
		`name="post_type"`,
		`value="post"`,
		`placeholder="What&#8217;s on your mind?"`,
	} {
		if !strings.Contains(body, needle) {
			t.Fatalf("missing %q in dashboard", needle)
		}
	}
}

func TestQuickDraftSaveRequiresNonceP7134(t *testing.T) {
	c := bootOrSkip(t)
	ck := loginCookie(t, c)
	srv := &server.Server{WP: c, AbsPath: "/workspace/", Front: &server.Front{}, Admin: &admin.Handler{}}

	form := url.Values{}
	form.Set("action", "post-quickdraft-save")
	form.Set("post_title", "Nonce Reject Draft")
	form.Set("content", "no nonce")
	req := httptest.NewRequest(http.MethodPost, "/wp-admin/post.php", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(ck)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "Unable to submit this form") {
		t.Fatalf("expected nonce failure notice: %q", rec.Body.String())
	}
}

func TestQuickDraftSaveWithNonceP7134(t *testing.T) {
	c := bootOrSkip(t)
	ck := loginCookie(t, c)
	srv := &server.Server{WP: c, AbsPath: "/workspace/", Front: &server.Front{}, Admin: &admin.Handler{}}

	req := httptest.NewRequest(http.MethodGet, "/wp-admin/", nil)
	req.AddCookie(ck)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	m := regexp.MustCompile(`name="_wpnonce" value="([^"]+)"`).FindStringSubmatch(rec.Body.String())
	if len(m) < 2 {
		t.Fatalf("nonce not found in dashboard")
	}
	nonce := m[1]

	form := url.Values{}
	form.Set("action", "post-quickdraft-save")
	form.Set("_wpnonce", nonce)
	form.Set("post_title", "Quick Draft Nonce OK")
	form.Set("content", "Hello with nonce")
	req = httptest.NewRequest(http.MethodPost, "/wp-admin/post.php", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(ck)
	rec = httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	body := rec.Body.String()
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%q", rec.Code, body)
	}
	if !strings.Contains(body, "Draft created successfully") {
		t.Fatalf("expected success: %q", body)
	}
	if !strings.Contains(body, `role="alert"`) {
		t.Fatalf("expected role=alert notice: %q", body)
	}
}

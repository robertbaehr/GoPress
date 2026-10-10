package admin_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/admin"
	"github.com/robertbaehr/GoPress/internal/config"
	"github.com/robertbaehr/GoPress/internal/login"
	"github.com/robertbaehr/GoPress/internal/server"
	"github.com/robertbaehr/GoPress/internal/wp"
)

func bootOrSkip(t *testing.T) *wp.Context {
	t.Helper()
	cfg := &config.Config{
		DBName: "gopress_go", DBUser: "gopress", DBPassword: "gopress",
		DBHost: "127.0.0.1:3306", DBCharset: "utf8mb4", TablePrefix: "wp_",
		ABSPATH:        "/workspace/",
		AuthKey:        "gopress-ref-auth-key-0001-fixed-for-oracle",
		SecureAuthKey:  "gopress-ref-secure-auth-key-0002-fixed",
		LoggedInKey:    "gopress-ref-logged-in-key-0003-fixed",
		NonceKey:       "gopress-ref-nonce-key-0004-fixed-oracle",
		AuthSalt:       "gopress-ref-auth-salt-0005-fixed-oracle",
		SecureAuthSalt: "gopress-ref-secure-auth-salt-0006-fixed",
		LoggedInSalt:   "gopress-ref-logged-in-salt-0007-fixed",
		NonceSalt:      "gopress-ref-nonce-salt-0008-fixed-oracle",
	}
	c, err := wp.Boot(context.Background(), cfg)
	if err != nil {
		t.Skipf("Boot unavailable: %v", err)
	}
	t.Cleanup(func() { _ = c.DB.Close() })
	return c
}

func loginCookie(t *testing.T, c *wp.Context) *http.Cookie {
	t.Helper()
	form := url.Values{}
	form.Set("log", "admin")
	form.Set("pwd", "password")
	form.Set("redirect_to", "/wp-admin/")
	req := httptest.NewRequest(http.MethodPost, "/wp-login.php", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	rc := c.ForRequest(rec, req)
	(&login.Handler{}).Handle(rec, req, rc)
	if rec.Code != http.StatusFound {
		t.Fatalf("login status=%d body=%q", rec.Code, rec.Body.String())
	}
	for _, ck := range rec.Result().Cookies() {
		if strings.HasPrefix(ck.Name, "wordpress_logged_in_") {
			return ck
		}
	}
	t.Fatal("no logged_in cookie")
	return nil
}

func TestAdminRedirectsAnonymous(t *testing.T) {
	c := bootOrSkip(t)
	srv := &server.Server{WP: c, AbsPath: "/workspace/", Front: &server.Front{}}
	for _, path := range []string{
		"/wp-admin/edit.php",
		"/wp-admin/options-general.php",
		"/wp-admin/plugins.php",
		"/wp-admin/profile.php",
		"/wp-admin/upload.php",
		"/wp-admin/edit-comments.php",
		"/wp-admin/themes.php",
		"/wp-admin/users.php",
		"/wp-admin/site-editor.php",
		"/wp-admin/nav-menus.php",
		"/wp-admin/customize.php",
	} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		srv.ServeHTTP(rec, req)
		if rec.Code != http.StatusFound {
			t.Fatalf("%s status=%d", path, rec.Code)
		}
		loc := rec.Header().Get("Location")
		if !strings.Contains(loc, "wp-login.php") || !strings.Contains(loc, "redirect_to=") {
			t.Fatalf("%s Location=%q", path, loc)
		}
	}
}

func TestAdminScreensLoggedIn(t *testing.T) {
	c := bootOrSkip(t)
	ck := loginCookie(t, c)
	srv := &server.Server{WP: c, AbsPath: "/workspace/", Front: &server.Front{}, Admin: &admin.Handler{}}

	cases := []struct {
		path string
		want string
	}{
		{"/wp-admin/edit.php", "wp-list-table"},
		{"/wp-admin/options-general.php", `name="blogname"`},
		{"/wp-admin/plugins.php", "Plugins"},
		{"/wp-admin/profile.php", `id="your-profile"`},
		{"/wp-admin/upload.php", "Media Library"},
		{"/wp-admin/edit-comments.php", "Comments"},
		{"/wp-admin/themes.php", "Themes"},
		{"/wp-admin/users.php", "Users"},
		{"/wp-admin/site-editor.php", "Site Editor"},
		{"/wp-admin/nav-menus.php", "menu-management"},
		{"/wp-admin/", "At a Glance"},
	}
	for _, tc := range cases {
		req := httptest.NewRequest(http.MethodGet, tc.path, nil)
		req.AddCookie(ck)
		rec := httptest.NewRecorder()
		srv.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("%s status=%d body=%q", tc.path, rec.Code, rec.Body.String())
		}
		if !strings.Contains(rec.Body.String(), tc.want) {
			t.Fatalf("%s missing %q in %q", tc.path, tc.want, rec.Body.String())
		}
	}
}

func TestCustomizerOutOfScope(t *testing.T) {
	c := bootOrSkip(t)
	ck := loginCookie(t, c)
	srv := &server.Server{WP: c, AbsPath: "/workspace/", Admin: &admin.Handler{}}
	req := httptest.NewRequest(http.MethodGet, "/wp-admin/customize.php", nil)
	req.AddCookie(ck)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, "out of scope") {
		t.Fatalf("expected out-of-scope message, got %q", body)
	}
	if !strings.Contains(body, "site-editor.php") {
		t.Fatalf("expected site editor link, got %q", body)
	}
}

func TestEditPostsListsTitles(t *testing.T) {
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
	if !strings.Contains(body, "Hello world!") && !strings.Contains(body, "Fixture Post") {
		t.Fatalf("expected post titles in table, got %q", body)
	}
	for _, want := range []string{"column-author", "column-date", "column-status", "Published", "admin"} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected %q in edit.php table, got %q", want, body)
		}
	}
}

func TestUploadListsMedia(t *testing.T) {
	c := bootOrSkip(t)
	ck := loginCookie(t, c)
	srv := &server.Server{WP: c, AbsPath: "/workspace/", Admin: &admin.Handler{}}
	req := httptest.NewRequest(http.MethodGet, "/wp-admin/upload.php", nil)
	req.AddCookie(ck)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, "sample-photo") && !strings.Contains(body, "wp-list-table") {
		t.Fatalf("expected media list, got %q", body)
	}
}

func TestUsersListsAdmin(t *testing.T) {
	c := bootOrSkip(t)
	ck := loginCookie(t, c)
	srv := &server.Server{WP: c, AbsPath: "/workspace/", Admin: &admin.Handler{}}
	req := httptest.NewRequest(http.MethodGet, "/wp-admin/users.php", nil)
	req.AddCookie(ck)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "admin") {
		t.Fatalf("expected admin user, got %q", rec.Body.String())
	}
}

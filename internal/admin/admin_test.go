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
	"github.com/robertbaehr/GoPress/internal/plugins"
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
		"/wp-admin/options-writing.php",
		"/wp-admin/options-reading.php",
		"/wp-admin/options-discussion.php",
		"/wp-admin/options-media.php",
		"/wp-admin/plugins.php",
		"/wp-admin/profile.php",
		"/wp-admin/upload.php",
		"/wp-admin/edit-comments.php",
		"/wp-admin/themes.php",
		"/wp-admin/users.php",
		"/wp-admin/user-new.php",
		"/wp-admin/site-editor.php",
		"/wp-admin/nav-menus.php",
		"/wp-admin/customize.php",
		"/wp-admin/post.php?post=1&action=edit",
		"/wp-admin/revision.php",
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
		{"/wp-admin/options-writing.php", `name="default_category"`},
		{"/wp-admin/options-reading.php", `name="posts_per_page"`},
		{"/wp-admin/options-discussion.php", `name="default_comment_status"`},
		{"/wp-admin/options-media.php", `name="thumbnail_size_w"`},
		{"/wp-admin/plugins.php", "Plugins"},
		{"/wp-admin/profile.php", `id="your-profile"`},
		{"/wp-admin/upload.php", "Media Library"},
		{"/wp-admin/edit-comments.php", "Comments"},
		{"/wp-admin/themes.php", "Theme Details"},
		{"/wp-admin/users.php", "Users"},
		{"/wp-admin/user-new.php", `id="createuser"`},
		{"/wp-admin/site-editor.php", "Site Editor"},
		{"/wp-admin/nav-menus.php", "menu-management"},
		{"/wp-admin/post.php?post=1&action=edit", `id="title"`},
		{"/wp-admin/revision.php?revision=1", "Compare Revisions"},
		{"/wp-admin/", "At a Glance"},
	}
	chromed := map[string]bool{
		"/wp-admin/":                    true,
		"/wp-admin/edit.php":            true,
		"/wp-admin/options-general.php": true,
		"/wp-admin/plugins.php":         true,
		"/wp-admin/profile.php":         true,
	}
	for _, tc := range cases {
		req := httptest.NewRequest(http.MethodGet, tc.path, nil)
		req.AddCookie(ck)
		rec := httptest.NewRecorder()
		srv.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("%s status=%d body=%q", tc.path, rec.Code, rec.Body.String())
		}
		body := rec.Body.String()
		if !strings.Contains(body, tc.want) {
			t.Fatalf("%s missing %q in %q", tc.path, tc.want, body)
		}
		if chromed[tc.path] {
			if !strings.Contains(body, `id="wpadminbar"`) || !strings.Contains(body, `id="adminmenumain"`) || !strings.Contains(body, `id="wpfooter"`) {
				t.Fatalf("%s missing shared admin chrome: %q", tc.path, body[:min(400, len(body))])
			}
			if !strings.Contains(body, "load-styles.php") || !strings.Contains(body, "colors.min.css") {
				t.Fatalf("%s missing admin CSS links: %q", tc.path, body[:min(600, len(body))])
			}
			if ct := rec.Header().Get("Content-Type"); ct != "text/html; charset=UTF-8" {
				t.Fatalf("%s Content-Type=%q", tc.path, ct)
			}
			if sc := rec.Header().Values("Set-Cookie"); len(sc) < 2 {
				t.Fatalf("%s expected wp-settings Set-Cookie headers, got %#v", tc.path, sc)
			}
		}
	}
}

func TestLinkManagerOutOfScope(t *testing.T) {
	c := bootOrSkip(t)
	ck := loginCookie(t, c)
	srv := &server.Server{WP: c, AbsPath: "/workspace/", Admin: &admin.Handler{}}
	for _, path := range []string{"/wp-admin/link-manager.php", "/wp-admin/link-add.php", "/wp-admin/link.php"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.AddCookie(ck)
		rec := httptest.NewRecorder()
		srv.ServeHTTP(rec, req)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("%s status=%d body=%q", path, rec.Code, rec.Body.String())
		}
		if !strings.Contains(rec.Body.String(), "out of scope") {
			t.Fatalf("%s expected out-of-scope message, got %q", path, rec.Body.String())
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

func TestThemeInstallOutOfScope(t *testing.T) {
	c := bootOrSkip(t)
	ck := loginCookie(t, c)
	srv := &server.Server{WP: c, AbsPath: "/workspace/", Admin: &admin.Handler{}}
	req := httptest.NewRequest(http.MethodGet, "/wp-admin/theme-install.php", nil)
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
	if !strings.Contains(body, "themes.php") {
		t.Fatalf("expected themes screen link, got %q", body)
	}
}

func TestThemesScreenDetails(t *testing.T) {
	c := bootOrSkip(t)
	ck := loginCookie(t, c)
	srv := &server.Server{WP: c, AbsPath: "/workspace/", Admin: &admin.Handler{}}
	req := httptest.NewRequest(http.MethodGet, "/wp-admin/themes.php", nil)
	req.AddCookie(ck)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	for _, want := range []string{
		"Themes",
		"theme-browser",
		"Theme Details",
		"theme-overlay",
		"theme-name",
		"theme-description",
		"Version:",
		"data-slug=",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing %q in themes.php", want)
		}
	}
}

func TestPluginInstallOutOfScope(t *testing.T) {
	c := bootOrSkip(t)
	ck := loginCookie(t, c)
	srv := &server.Server{WP: c, AbsPath: "/workspace/", Admin: &admin.Handler{}}
	req := httptest.NewRequest(http.MethodGet, "/wp-admin/plugin-install.php", nil)
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
	if !strings.Contains(body, "plugins.php") {
		t.Fatalf("expected plugins screen link, got %q", body)
	}
}

func TestUpdateCoreOutOfScope(t *testing.T) {
	c := bootOrSkip(t)
	ck := loginCookie(t, c)
	srv := &server.Server{WP: c, AbsPath: "/workspace/", Admin: &admin.Handler{}}
	req := httptest.NewRequest(http.MethodGet, "/wp-admin/update-core.php", nil)
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
	if !strings.Contains(body, "update-core.php") {
		t.Fatalf("expected update-core mention, got %q", body)
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
	body := rec.Body.String()
	if !strings.Contains(body, "admin") {
		t.Fatalf("expected admin user, got %q", body)
	}
	for _, want := range []string{
		"column-username",
		"column-name",
		"column-email",
		"column-role",
		"column-posts",
		"Administrator",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing %q in users table: %q", want, body)
		}
	}
}

func TestUserNewFormStub(t *testing.T) {
	c := bootOrSkip(t)
	ck := loginCookie(t, c)
	srv := &server.Server{WP: c, AbsPath: "/workspace/", Admin: &admin.Handler{}}
	req := httptest.NewRequest(http.MethodGet, "/wp-admin/user-new.php", nil)
	req.AddCookie(ck)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	for _, want := range []string{
		`id="createuser"`,
		`name="user_login"`,
		`name="email"`,
		`name="role"`,
		"Add New User",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing %q in %q", want, body)
		}
	}
}

func TestPluginsListsHelloDollyHeaders(t *testing.T) {
	c := bootOrSkip(t)
	ck := loginCookie(t, c)
	srv := &server.Server{WP: c, AbsPath: "/workspace/", Admin: &admin.Handler{}}
	req := httptest.NewRequest(http.MethodGet, "/wp-admin/plugins.php", nil)
	req.AddCookie(ck)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Hello Dolly") {
		t.Fatalf("expected Hello Dolly from plugin headers, got %q", body)
	}
	if !strings.Contains(body, "Matt Mullenweg") {
		t.Fatalf("expected author from headers, got %q", body)
	}
	if !strings.Contains(body, "Version 1.7.2") {
		t.Fatalf("expected Version from headers, got %q", body)
	}
	if !strings.Contains(body, "Visit plugin site") || !strings.Contains(body, "wordpress.org/plugins/hello-dolly") {
		t.Fatalf("expected Plugin URI link, got %q", body)
	}
	if !strings.Contains(body, "plugin-version-author-uri") {
		t.Fatalf("expected plugin meta line, got %q", body)
	}
	if !strings.Contains(body, "action=activate") {
		t.Fatalf("expected activate link, got %q", body)
	}
}

func TestPluginsActivateDeactivateHelloDolly(t *testing.T) {
	c := bootOrSkip(t)
	t.Cleanup(func() {
		_, _ = c.Options.UpdateOption("active_plugins", `a:0:{}`, "yes")
	})
	_, _ = c.Options.UpdateOption("active_plugins", `a:0:{}`, "yes")

	ck := loginCookie(t, c)
	srv := &server.Server{WP: c, AbsPath: "/workspace/", Admin: &admin.Handler{}}

	req := httptest.NewRequest(http.MethodGet, "/wp-admin/plugins.php", nil)
	req.AddCookie(ck)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("list status=%d", rec.Code)
	}
	body := rec.Body.String()
	activateHref := extractHref(body, "action=activate")
	if activateHref == "" {
		t.Fatalf("missing activate href in %q", body)
	}

	req = httptest.NewRequest(http.MethodGet, activateHref, nil)
	req.AddCookie(ck)
	rec = httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusFound {
		t.Fatalf("activate status=%d body=%q", rec.Code, rec.Body.String())
	}
	if !plugins.IsPluginActive(c, "hello.php") {
		t.Fatal("expected hello.php active after activate")
	}

	req = httptest.NewRequest(http.MethodGet, "/wp-admin/plugins.php", nil)
	req.AddCookie(ck)
	rec = httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	body = rec.Body.String()
	if !strings.Contains(body, `class="active"`) {
		t.Fatalf("expected active row: %q", body)
	}
	deactivateHref := extractHref(body, "action=deactivate")
	if deactivateHref == "" {
		t.Fatalf("missing deactivate href in %q", body)
	}

	u, err := url.Parse(deactivateHref)
	if err != nil {
		t.Fatal(err)
	}
	form := url.Values{}
	form.Set("action", u.Query().Get("action"))
	form.Set("plugin", u.Query().Get("plugin"))
	form.Set("_wpnonce", u.Query().Get("_wpnonce"))
	req = httptest.NewRequest(http.MethodPost, "/wp-admin/plugins.php", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(ck)
	rec = httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusFound {
		t.Fatalf("deactivate status=%d body=%q", rec.Code, rec.Body.String())
	}
	if plugins.IsPluginActive(c, "hello.php") {
		t.Fatal("expected hello.php inactive after deactivate")
	}
}

func extractHref(body, needle string) string {
	idx := strings.Index(body, needle)
	if idx < 0 {
		return ""
	}
	// Walk back to href="
	start := strings.LastIndex(body[:idx], `href="`)
	if start < 0 {
		return ""
	}
	start += len(`href="`)
	end := strings.Index(body[start:], `"`)
	if end < 0 {
		return ""
	}
	href := body[start : start+end]
	return strings.ReplaceAll(href, "&amp;", "&")
}

func TestProfileUpdateDisplayName(t *testing.T) {
	c := bootOrSkip(t)
	ck := loginCookie(t, c)
	srv := &server.Server{WP: c, AbsPath: "/workspace/", Admin: &admin.Handler{}}

	user, err := c.GetUserBy("login", "admin")
	if err != nil || user == nil {
		t.Fatalf("admin: %v", err)
	}
	original := user.DisplayName
	t.Cleanup(func() {
		_ = c.UpdateDisplayName(user.ID, original)
	})

	req := httptest.NewRequest(http.MethodGet, "/wp-admin/profile.php", nil)
	req.AddCookie(ck)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET status=%d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `method="post"`) || !strings.Contains(body, `name="_wpnonce"`) {
		t.Fatalf("expected POST form with nonce: %q", body[:min(500, len(body))])
	}
	nonceStart := strings.Index(body, `name="_wpnonce" value="`)
	if nonceStart < 0 {
		t.Fatal("nonce not found")
	}
	nonceStart += len(`name="_wpnonce" value="`)
	nonceEnd := strings.Index(body[nonceStart:], `"`)
	nonce := body[nonceStart : nonceStart+nonceEnd]

	wantName := "GoPress Profile Test"
	form := url.Values{}
	form.Set("_wpnonce", nonce)
	form.Set("display_name", wantName)
	req = httptest.NewRequest(http.MethodPost, "/wp-admin/profile.php", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(ck)
	rec = httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("POST status=%d body=%q", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "Profile updated") {
		t.Fatalf("missing updated notice: %q", rec.Body.String()[:min(400, rec.Body.Len())])
	}
	if !strings.Contains(rec.Body.String(), wantName) {
		t.Fatalf("missing new display name in form: %q", rec.Body.String()[:min(600, rec.Body.Len())])
	}
	fresh, err := c.GetUserBy("id", user.ID)
	if err != nil || fresh == nil {
		t.Fatalf("reload user: %v", err)
	}
	if fresh.DisplayName != wantName {
		t.Fatalf("display_name=%q want %q", fresh.DisplayName, wantName)
	}
}

func TestNavMenusLocationsSave(t *testing.T) {
	c := bootOrSkip(t)
	ck := loginCookie(t, c)
	t.Cleanup(func() {
		_ = wp.SetNavMenuLocation(c, "primary", 0)
		_ = wp.SetNavMenuLocation(c, "footer", 0)
	})
	h := &admin.Handler{}

	req := httptest.NewRequest(http.MethodGet, "/wp-admin/nav-menus.php?action=locations", nil)
	req.AddCookie(ck)
	rec := httptest.NewRecorder()
	rc := c.ForRequest(rec, req)
	if !h.Handle(rec, req, rc, "/wp-admin/nav-menus.php") {
		t.Fatal("GET locations not handled")
	}
	body := rec.Body.String()
	for _, want := range []string{
		"menu-locations-table",
		`name="menu-locations[primary]"`,
		`name="menu-locations[footer]"`,
		"Manage Locations",
		"Save Changes",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing %q in %q", want, body[:min(800, len(body))])
		}
	}

	form := url.Values{}
	form.Set("menu-locations[primary]", "37")
	form.Set("menu-locations[footer]", "0")
	form.Set("nav-menu-locations", "Save Changes")
	req = httptest.NewRequest(http.MethodPost, "/wp-admin/nav-menus.php?action=locations", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(ck)
	rec = httptest.NewRecorder()
	rc = c.ForRequest(rec, req)
	if !h.Handle(rec, req, rc, "/wp-admin/nav-menus.php") {
		t.Fatal("POST locations not handled")
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "Menu locations updated") {
		t.Fatalf("missing save notice: %q", rec.Body.String()[:min(500, rec.Body.Len())])
	}
	if !strings.Contains(rec.Body.String(), `value="37" selected="selected"`) {
		t.Fatalf("expected primary selected: %q", rec.Body.String()[:min(1200, rec.Body.Len())])
	}
	locs := wp.GetNavMenuLocations(c)
	if locs["primary"] != 37 {
		t.Fatalf("primary location=%v want 37", locs)
	}
}

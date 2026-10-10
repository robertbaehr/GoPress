package plugins_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/admin"
	"github.com/robertbaehr/GoPress/internal/config"
	"github.com/robertbaehr/GoPress/internal/login"
	"github.com/robertbaehr/GoPress/internal/plugins"
	"github.com/robertbaehr/GoPress/internal/server"
	"github.com/robertbaehr/GoPress/internal/wp"
	"github.com/robertbaehr/GoPress/plugins/hellodolly"
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

func TestHelloDollyActivateHooksAdminNotices(t *testing.T) {
	c := bootOrSkip(t)
	// Ensure plugin is active for this process even if option is empty.
	hellodolly.Plugin{}.Activate(c)
	html := c.CaptureAdminNotices()
	if !strings.Contains(html, `id="dolly"`) {
		t.Fatalf("expected dolly notice, got %q", html)
	}
	lyric := hellodolly.RandomLyric()
	if lyric == "" {
		t.Fatal("empty lyric")
	}
}

func TestScanMUPluginFiles(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "alpha.php"), []byte("<?php"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(dir, "beta"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "beta", "beta.php"), []byte("<?php"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "readme.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	files, err := plugins.ScanMUPluginFiles(dir)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]bool{"alpha.php": true, "beta/beta.php": true}
	if len(files) != 2 {
		t.Fatalf("files=%v", files)
	}
	for _, f := range files {
		if !want[f] {
			t.Fatalf("unexpected %q in %v", f, files)
		}
	}
}

type muStub struct {
	file string
	hit  *bool
}

func (m muStub) File() string { return m.file }
func (m muStub) Activate(c *wp.Context) {
	if m.hit != nil {
		*m.hit = true
	}
}

func TestLoadMUPluginsActivatesRegistered(t *testing.T) {
	c := bootOrSkip(t)
	root := t.TempDir()
	mu := filepath.Join(root, "wp-content", "mu-plugins")
	if err := os.MkdirAll(mu, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(mu, "mu-test.php"), []byte("<?php"), 0o644); err != nil {
		t.Fatal(err)
	}
	c.Config.ABSPATH = root + string(os.PathSeparator)
	hit := false
	plugins.Register(muStub{file: "mu-test.php", hit: &hit})
	if err := plugins.LoadMUPlugins(c); err != nil {
		t.Fatal(err)
	}
	if !hit {
		t.Fatal("expected mu-test.php shim to Activate")
	}
}

func TestLoadActiveFromOption(t *testing.T) {
	c := bootOrSkip(t)
	// Serialized active_plugins with hello.php
	serialized := `a:1:{i:0;s:9:"hello.php";}`
	if _, err := c.Options.UpdateOption("active_plugins", serialized, "yes"); err != nil {
		t.Fatalf("UpdateOption: %v", err)
	}
	t.Cleanup(func() {
		_, _ = c.Options.UpdateOption("active_plugins", `a:0:{}`, "yes")
	})
	files, err := plugins.ActivePluginFiles(c)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 || files[0] != "hello.php" {
		t.Fatalf("files=%v", files)
	}
}

func TestDashboardShowsDollyWhenActive(t *testing.T) {
	c := bootOrSkip(t)
	serialized := `a:1:{i:0;s:9:"hello.php";}`
	if _, err := c.Options.UpdateOption("active_plugins", serialized, "yes"); err != nil {
		t.Fatalf("UpdateOption: %v", err)
	}
	t.Cleanup(func() {
		_, _ = c.Options.UpdateOption("active_plugins", `a:0:{}`, "yes")
	})
	// Re-activate on this context (Boot already ran with prior option).
	if err := plugins.LoadActive(c); err != nil {
		t.Fatal(err)
	}

	form := url.Values{}
	form.Set("log", "admin")
	form.Set("pwd", "password")
	form.Set("redirect_to", "/wp-admin/")
	req := httptest.NewRequest(http.MethodPost, "/wp-login.php", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	rc := c.ForRequest(rec, req)
	(&login.Handler{}).Handle(rec, req, rc)
	var ck *http.Cookie
	for _, cki := range rec.Result().Cookies() {
		if strings.HasPrefix(cki.Name, "wordpress_logged_in_") {
			ck = cki
			break
		}
	}
	if ck == nil {
		t.Fatal("no login cookie")
	}

	srv := &server.Server{WP: c, AbsPath: "/workspace/", Admin: &admin.Handler{}}
	req = httptest.NewRequest(http.MethodGet, "/wp-admin/", nil)
	req.AddCookie(ck)
	rec = httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `id="dolly"`) {
		t.Fatalf("expected dolly in dashboard, got %q", rec.Body.String())
	}
}

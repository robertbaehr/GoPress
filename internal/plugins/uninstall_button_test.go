package plugins_test

import (
	"context"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/config"
	"github.com/robertbaehr/GoPress/internal/plugins"
	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestUninstallActionLinkForUninstallable(t *testing.T) {
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
		t.Skip(err)
	}
	defer c.DB.Close()

	user, err := c.GetUserBy("login", "admin")
	if err != nil || user == nil {
		t.Fatalf("admin: %v", err)
	}
	c.CurrentUser = user

	file := "hello.php"
	_ = plugins.DeactivatePlugin(c, file)
	if err := plugins.RegisterUninstallHook(c, file, nil); err != nil {
		t.Fatalf("RegisterUninstallHook: %v", err)
	}
	t.Cleanup(func() {
		_ = plugins.UninstallPlugin(c, file)
	})

	p := plugins.PluginData{File: file, Name: "Hello Dolly"}
	actions := plugins.ActionLinks(c, p, "inactive")
	html := plugins.FormatActionLinksHTML(actions)
	for _, want := range []string{
		`class="uninstall"`,
		`action=uninstall`,
		`>Uninstall</a>`,
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("missing uninstall button %q in %q", want, html)
		}
	}

	// Active plugins do not show Uninstall.
	if err := plugins.ActivatePlugin(c, file); err != nil {
		t.Fatalf("activate: %v", err)
	}
	actions = plugins.ActionLinks(c, p, "active")
	html = plugins.FormatActionLinksHTML(actions)
	if strings.Contains(html, `class="uninstall"`) {
		t.Fatalf("active plugin should not show Uninstall: %q", html)
	}
	_ = plugins.DeactivatePlugin(c, file)
}

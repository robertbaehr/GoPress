package plugins_test

import (
	"context"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/config"
	"github.com/robertbaehr/GoPress/internal/plugins"
	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestActionLinksFilter(t *testing.T) {
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

	c.Hooks.AddFilter("plugin_action_links", func(value any, args ...any) any {
		m, _ := value.(map[string]string)
		m["settings"] = `<a href="/wp-admin/options-general.php">Settings</a>`
		return m
	}, 10, 4)
	c.Hooks.AddFilter("plugin_action_links_hello.php", func(value any, args ...any) any {
		m, _ := value.(map[string]string)
		m["dolly"] = `<a href="#">Dolly</a>`
		return m
	}, 10, 4)

	p := plugins.PluginData{File: "hello.php", Name: "Hello Dolly"}
	actions := plugins.ActionLinks(c, p, "inactive")
	html := plugins.FormatActionLinksHTML(actions)
	for _, want := range []string{`class="activate"`, `class="settings"`, `class="dolly"`, "Settings", "Dolly"} {
		if !strings.Contains(html, want) {
			t.Fatalf("missing %q in %q", want, html)
		}
	}
}

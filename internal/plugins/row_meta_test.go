package plugins_test

import (
	"context"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/config"
	"github.com/robertbaehr/GoPress/internal/plugins"
	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestRowMetaFilter(t *testing.T) {
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

	c.Hooks.AddFilter("plugin_row_meta", func(value any, args ...any) any {
		meta, _ := value.([]string)
		meta = append(meta, `<a href="https://example.test/docs">View details</a>`)
		return meta
	}, 10, 4)

	after := false
	c.Hooks.AddAction("after_plugin_row_meta", func(args ...any) {
		after = true
	}, 10, 2)

	p := plugins.PluginData{
		File:      "hello.php",
		Name:      "Hello Dolly",
		Version:   "1.7.2",
		Author:    "Matt Mullenweg",
		AuthorURI: "http://ma.tt/",
		PluginURI: "http://wordpress.org/plugins/hello-dolly/",
	}
	html := plugins.FormatRowMetaHTML(plugins.RowMeta(c, p, "all"))
	for _, want := range []string{
		"Version 1.7.2",
		"By ",
		"Matt Mullenweg",
		"Visit plugin site",
		"View details",
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("missing %q in %q", want, html)
		}
	}
	if !after {
		t.Fatal("expected after_plugin_row_meta action")
	}
}

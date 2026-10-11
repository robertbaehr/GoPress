package plugins_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/config"
	"github.com/robertbaehr/GoPress/internal/plugins"
)

func TestPluginsURLBaseP946(t *testing.T) {
	c := bootOrSkip(t)
	c.Options.Cache.Set("siteurl", "http://example.test", "options")
	got := plugins.PluginsURL(c, "", "")
	if !strings.HasSuffix(got, "/wp-content/plugins") {
		t.Fatalf("base=%q", got)
	}
}

func TestPluginsURLPluginFolderAndPathP946(t *testing.T) {
	c := bootOrSkip(t)
	c.Options.Cache.Set("siteurl", "http://example.test", "options")
	root := t.TempDir()
	c.Config = &config.Config{ABSPATH: root + string(filepath.Separator)}
	// Keep Options from boot; override ABSPATH for PluginDir resolution.
	file := filepath.Join(plugins.PluginDir(c), "hello-dolly", "hello.php")
	got := plugins.PluginsURL(c, "css/style.css", file)
	if !strings.Contains(got, "/wp-content/plugins/hello-dolly/css/style.css") {
		t.Fatalf("got=%q", got)
	}
	dir := plugins.PluginDirURL(c, file)
	if !strings.HasSuffix(dir, "/hello-dolly/") {
		t.Fatalf("PluginDirURL=%q", dir)
	}
}

func TestPluginsURLMUPluginP946(t *testing.T) {
	c := bootOrSkip(t)
	c.Options.Cache.Set("siteurl", "http://example.test", "options")
	root := t.TempDir()
	c.Config = &config.Config{ABSPATH: root + string(filepath.Separator)}
	file := filepath.Join(plugins.MUPluginDir(c), "mu-loader.php")
	got := plugins.PluginsURL(c, "", file)
	if !strings.Contains(got, "/wp-content/mu-plugins") {
		t.Fatalf("mu url=%q", got)
	}
	// Single-file mu-plugin: no extra folder segment.
	if strings.Contains(got, "mu-loader") {
		t.Fatalf("single-file should not append basename: %q", got)
	}
}

func TestPluginsURLFilterP946(t *testing.T) {
	c := bootOrSkip(t)
	c.Hooks.AddFilter("plugins_url", func(value any, _ ...any) any {
		return "http://cdn.example/plugins"
	}, 10, 3)
	if got := plugins.PluginsURL(c, "x.js", ""); got != "http://cdn.example/plugins" {
		t.Fatalf("filter=%q", got)
	}
}

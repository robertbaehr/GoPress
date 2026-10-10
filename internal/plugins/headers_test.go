package plugins_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/robertbaehr/GoPress/internal/plugins"
)

func TestGetPluginDataHelloDolly(t *testing.T) {
	path := "/workspace/wp-content/plugins/hello.php"
	if _, err := os.Stat(path); err != nil {
		t.Skip(err)
	}
	data, err := plugins.GetPluginData(path)
	if err != nil {
		t.Fatal(err)
	}
	if data.Name != "Hello Dolly" {
		t.Fatalf("Name=%q", data.Name)
	}
	if data.Version != "1.7.2" {
		t.Fatalf("Version=%q", data.Version)
	}
	if data.Author != "Matt Mullenweg" {
		t.Fatalf("Author=%q", data.Author)
	}
	if data.TextDomain != "hello-dolly" {
		t.Fatalf("TextDomain=%q", data.TextDomain)
	}
	if data.PluginURI == "" || data.Description == "" {
		t.Fatalf("missing uri/desc: %+v", data)
	}
}

func TestGetPluginDataFromTemp(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sample.php")
	content := `<?php
/**
 * Plugin Name: Sample Plugin
 * Version: 0.1.0
 * Author: GoPress
 * Network: true
 */
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	data, err := plugins.GetPluginData(path)
	if err != nil {
		t.Fatal(err)
	}
	if data.Name != "Sample Plugin" || data.Version != "0.1.0" || !data.Network {
		t.Fatalf("%+v", data)
	}
}

func TestListMUPluginsHeaders(t *testing.T) {
	c := bootOrSkip(t)
	root := t.TempDir()
	mu := filepath.Join(root, "wp-content", "mu-plugins")
	if err := os.MkdirAll(mu, 0o755); err != nil {
		t.Fatal(err)
	}
	named := `<?php
/**
 * Plugin Name: MU Fixture
 * Version: 2.0
 * Description: Must-use test plugin
 */
`
	if err := os.WriteFile(filepath.Join(mu, "mu-fixture.php"), []byte(named), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(mu, "index.php"), []byte("<?php\n// Silence is golden.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(mu, "subdir"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(mu, "subdir", "nested.php"), []byte("<?php\n/** Plugin Name: Nested */\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	c.Config.ABSPATH = root + string(os.PathSeparator)
	list, err := plugins.ListMUPlugins(c)
	if err != nil {
		t.Fatal(err)
	}
	byFile := map[string]plugins.PluginData{}
	for _, p := range list {
		byFile[p.File] = p
	}
	if _, ok := byFile["subdir/nested.php"]; ok {
		t.Fatalf("get_mu_plugins must not include subdir plugins: %+v", list)
	}
	fix, ok := byFile["mu-fixture.php"]
	if !ok || fx.Name != "MU Fixture" || fx.Version != "2.0" {
		t.Fatalf("fixture: %+v list=%+v", fx, list)
	}
	idx, ok := byFile["index.php"]
	if !ok || idx.Name != "index.php" {
		t.Fatalf("index.php should default Name to basename: %+v", idx)
	}
}

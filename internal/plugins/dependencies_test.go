package plugins_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/robertbaehr/GoPress/internal/config"
	"github.com/robertbaehr/GoPress/internal/plugins"
	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestSanitizeDependencySlugs(t *testing.T) {
	got := plugins.SanitizeDependencySlugs(" hello-dolly, WooCommerce, bad slug!,hello-dolly ")
	if len(got) != 2 || got[0] != "hello-dolly" || got[1] != "woocommerce" {
		t.Fatalf("got %#v", got)
	}
	if got := plugins.SanitizeDependencySlugs(""); len(got) != 0 {
		t.Fatalf("empty: %#v", got)
	}
}

func TestConvertToSlug(t *testing.T) {
	if plugins.ConvertToSlug("hello-dolly/hello.php") != "hello-dolly" {
		t.Fatal(plugins.ConvertToSlug("hello-dolly/hello.php"))
	}
	if plugins.ConvertToSlug("hello.php") != "hello" {
		t.Fatal(plugins.ConvertToSlug("hello.php"))
	}
}

func TestReadPluginDependencies(t *testing.T) {
	root := t.TempDir()
	plugDir := filepath.Join(root, "wp-content", "plugins")
	depDir := filepath.Join(plugDir, "dep-one")
	needDir := filepath.Join(plugDir, "needs-dep")
	if err := os.MkdirAll(depDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(needDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writePlugin := func(dir, name, requires string) {
		body := "<?php\n/**\n * Plugin Name: " + name + "\n"
		if requires != "" {
			body += " * Requires Plugins: " + requires + "\n"
		}
		body += " */\n"
		if err := os.WriteFile(filepath.Join(dir, name+".php"), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	writePlugin(depDir, "dep-one", "")
	writePlugin(needDir, "needs-dep", "dep-one, missing-plugin")

	c := &wp.Context{Config: &config.Config{ABSPATH: root}}
	pd, err := plugins.ReadPluginDependencies(c)
	if err != nil {
		t.Fatal(err)
	}
	needFile := "needs-dep/needs-dep.php"
	if !pd.HasDependencies(needFile) {
		t.Fatalf("expected dependencies for %s: %#v", needFile, pd.Dependencies)
	}
	deps := pd.GetDependencies(needFile)
	if len(deps) != 2 || deps[0] != "dep-one" || deps[1] != "missing-plugin" {
		t.Fatalf("deps=%#v", deps)
	}
	if pd.GetDependencyFilepath("dep-one") == "" {
		t.Fatal("expected dep-one installed")
	}
	if !pd.HasUnmetDependencies(needFile) {
		t.Fatal("expected unmet missing-plugin")
	}
	depsOf := pd.GetDependents("dep-one")
	if len(depsOf) != 1 || depsOf[0] != needFile {
		t.Fatalf("dependents=%#v", depsOf)
	}
}

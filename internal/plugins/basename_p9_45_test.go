package plugins_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/robertbaehr/GoPress/internal/config"
	"github.com/robertbaehr/GoPress/internal/plugins"
	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestPluginBasenameStripsPluginDir(t *testing.T) {
	plugins.ResetPluginPathsForTest()
	t.Cleanup(plugins.ResetPluginPathsForTest)

	root := t.TempDir()
	c := &wp.Context{Config: &config.Config{ABSPATH: root + string(filepath.Separator)}}
	abs := filepath.Join(plugins.PluginDir(c), "hello-dolly", "hello.php")
	got := plugins.PluginBasename(c, abs)
	if got != "hello-dolly/hello.php" {
		t.Fatalf("PluginBasename=%q want hello-dolly/hello.php", got)
	}
}

func TestPluginBasenameStripsMUPluginDir(t *testing.T) {
	plugins.ResetPluginPathsForTest()
	t.Cleanup(plugins.ResetPluginPathsForTest)

	root := t.TempDir()
	c := &wp.Context{Config: &config.Config{ABSPATH: root + string(filepath.Separator)}}
	abs := filepath.Join(plugins.MUPluginDir(c), "mu-loader.php")
	got := plugins.PluginBasename(c, abs)
	if got != "mu-loader.php" {
		t.Fatalf("PluginBasename=%q want mu-loader.php", got)
	}
}

func TestPluginBasenameAlreadyRelative(t *testing.T) {
	plugins.ResetPluginPathsForTest()
	t.Cleanup(plugins.ResetPluginPathsForTest)

	root := t.TempDir()
	c := &wp.Context{Config: &config.Config{ABSPATH: root + string(filepath.Separator)}}
	got := plugins.PluginBasename(c, "akismet/akismet.php")
	if got != "akismet/akismet.php" {
		t.Fatalf("PluginBasename=%q want akismet/akismet.php", got)
	}
}

func TestPluginBasenameSymlinkRemap(t *testing.T) {
	plugins.ResetPluginPathsForTest()
	t.Cleanup(plugins.ResetPluginPathsForTest)

	root := t.TempDir()
	c := &wp.Context{Config: &config.Config{ABSPATH: root + string(filepath.Separator)}}

	realTree := filepath.Join(root, "real-plugins", "sym-plugin")
	if err := os.MkdirAll(realTree, 0o755); err != nil {
		t.Fatal(err)
	}
	realFile := filepath.Join(realTree, "plugin.php")
	if err := os.WriteFile(realFile, []byte("<?php"), 0o644); err != nil {
		t.Fatal(err)
	}

	pluginRoot := plugins.PluginDir(c)
	if err := os.MkdirAll(pluginRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	linkDir := filepath.Join(pluginRoot, "sym-plugin")
	if err := os.Symlink(realTree, linkDir); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	linkFile := filepath.Join(linkDir, "plugin.php")

	if ok := plugins.RegisterPluginRealpath(c, linkFile); !ok {
		t.Fatal("RegisterPluginRealpath returned false")
	}
	// Resolve through the real path (as callers often do after realpath).
	resolved, err := filepath.EvalSymlinks(linkFile)
	if err != nil {
		t.Fatal(err)
	}
	got := plugins.PluginBasename(c, resolved)
	if got != "sym-plugin/plugin.php" {
		t.Fatalf("PluginBasename after remap=%q want sym-plugin/plugin.php (resolved=%q)", got, resolved)
	}
}

func TestRegisterPluginRealpathDirectChildFalse(t *testing.T) {
	plugins.ResetPluginPathsForTest()
	t.Cleanup(plugins.ResetPluginPathsForTest)

	root := t.TempDir()
	c := &wp.Context{Config: &config.Config{ABSPATH: root + string(filepath.Separator)}}
	pluginRoot := plugins.PluginDir(c)
	if err := os.MkdirAll(pluginRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(pluginRoot, "single-file.php")
	if err := os.WriteFile(file, []byte("<?php"), 0o644); err != nil {
		t.Fatal(err)
	}
	if plugins.RegisterPluginRealpath(c, file) {
		t.Fatal("expected false for file directly under PluginDir")
	}
}

func TestPluginDirPath(t *testing.T) {
	got := plugins.PluginDirPath("/var/www/wp-content/plugins/hello/hello.php")
	if got != "/var/www/wp-content/plugins/hello/" {
		t.Fatalf("PluginDirPath=%q", got)
	}
	got = plugins.PluginDirPath(`C:\plugins\hello\hello.php`)
	if got != "C:/plugins/hello/" {
		t.Fatalf("PluginDirPath windows=%q", got)
	}
}

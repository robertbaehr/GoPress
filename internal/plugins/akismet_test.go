package plugins_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/plugins"
)

func TestIsAkismetPlugin(t *testing.T) {
	if !plugins.IsAkismetPlugin("akismet/akismet.php") {
		t.Fatal("expected akismet/akismet.php")
	}
	if !plugins.IsAkismetPlugin("akismet.php") {
		t.Fatal("expected akismet.php")
	}
	if plugins.IsAkismetPlugin("hello.php") {
		t.Fatal("hello is not akismet")
	}
}

func TestActivatePluginRejectsAkismet(t *testing.T) {
	c := bootOrSkip(t)
	t.Cleanup(func() { _ = c.DB.Close() })
	err := plugins.ActivatePlugin(c, "akismet/akismet.php")
	if err == nil {
		t.Fatal("expected error activating akismet")
	}
	if !strings.Contains(err.Error(), "Akismet") {
		t.Fatalf("err=%v", err)
	}
}

func TestAkismetPresentOnDisk(t *testing.T) {
	c := bootOrSkip(t)
	t.Cleanup(func() { _ = c.DB.Close() })
	if plugins.AkismetPresentOnDisk(c) {
		t.Fatal("fixture should not ship akismet")
	}
	pluginsDir := filepath.Join(plugins.ContentDir(c), "plugins", "akismet")
	if err := os.MkdirAll(pluginsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(pluginsDir, "akismet.php")
	if err := os.WriteFile(path, []byte("<?php\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(pluginsDir) })
	if !plugins.AkismetPresentOnDisk(c) {
		t.Fatal("expected detection after drop-in")
	}
}

func TestAkismetNotBundledNotice(t *testing.T) {
	if !strings.Contains(plugins.AkismetNotBundledNotice, "not bundled") {
		t.Fatal("notice text")
	}
	c := bootOrSkip(t)
	t.Cleanup(func() { _ = c.DB.Close() })
	plugins.CheckAkismetGone(c)
	c.Hooks.DoAction("admin_notices")
}

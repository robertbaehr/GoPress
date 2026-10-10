package plugins_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/config"
	"github.com/robertbaehr/GoPress/internal/plugins"
	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestDetectUnsupportedDropIns(t *testing.T) {
	root := t.TempDir()
	content := filepath.Join(root, "wp-content")
	if err := os.MkdirAll(content, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"advanced-cache.php", "object-cache.php", "readme.txt"} {
		if err := os.WriteFile(filepath.Join(content, name), []byte("<?php"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	c := &wp.Context{Config: &config.Config{ABSPATH: root}, Hooks: wp.NewHooks()}
	found, err := plugins.DetectUnsupportedDropIns(c)
	if err != nil {
		t.Fatal(err)
	}
	if len(found) != 2 {
		t.Fatalf("found=%#v", found)
	}
	if found[0].File != "advanced-cache.php" || found[1].File != "object-cache.php" {
		t.Fatalf("order/names: %#v", found)
	}
	plugins.RegisterDropInNotices(c, found)
	html := c.CaptureAdminNotices()
	if !strings.Contains(html, "gopress-dropins") {
		t.Fatalf("missing notice: %q", html)
	}
	if !strings.Contains(html, "advanced-cache.php") || !strings.Contains(html, "object-cache.php") {
		t.Fatalf("missing filenames: %q", html)
	}
}

func TestDetectUnsupportedDropInsEmpty(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "wp-content"), 0o755); err != nil {
		t.Fatal(err)
	}
	c := &wp.Context{Config: &config.Config{ABSPATH: root}}
	found, err := plugins.DetectUnsupportedDropIns(c)
	if err != nil {
		t.Fatal(err)
	}
	if len(found) != 0 {
		t.Fatalf("expected none, got %#v", found)
	}
}

func TestKnownDropInsIncludesCoreFiles(t *testing.T) {
	want := map[string]bool{
		"advanced-cache.php": true,
		"db.php":             true,
		"object-cache.php":   true,
	}
	for _, d := range plugins.KnownDropIns {
		delete(want, d.File)
	}
	if len(want) != 0 {
		t.Fatalf("missing known drop-ins: %v", want)
	}
}

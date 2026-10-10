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

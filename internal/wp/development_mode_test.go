package wp_test

import (
	"os"
	"testing"

	"github.com/robertbaehr/GoPress/internal/config"
	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestGetDevelopmentModeDefaultEmpty(t *testing.T) {
	os.Unsetenv("WP_DEVELOPMENT_MODE")
	if got := wp.GetDevelopmentMode(nil); got != "" {
		t.Fatalf("nil context: got %q", got)
	}
	c := &wp.Context{}
	if got := wp.GetDevelopmentMode(c); got != "" {
		t.Fatalf("empty context: got %q", got)
	}
	c.Config = &config.Config{}
	if got := wp.GetDevelopmentMode(c); got != "" {
		t.Fatalf("empty config: got %q", got)
	}
	if wp.IsDevelopmentMode(c, "core") {
		t.Fatal("disabled mode should be false for core")
	}
}

func TestGetDevelopmentModeFromConfig(t *testing.T) {
	os.Unsetenv("WP_DEVELOPMENT_MODE")
	c := &wp.Context{Config: &config.Config{DevelopmentMode: "plugin"}}
	if got := wp.GetDevelopmentMode(c); got != "plugin" {
		t.Fatalf("got %q want plugin", got)
	}
	c.Config.DevelopmentMode = "THEME"
	if got := wp.GetDevelopmentMode(c); got != "theme" {
		t.Fatalf("got %q want theme", got)
	}
	c.Config.DevelopmentMode = "nope"
	if got := wp.GetDevelopmentMode(c); got != "" {
		t.Fatalf("invalid should fall back to empty, got %q", got)
	}
}

func TestIsDevelopmentModeAll(t *testing.T) {
	os.Unsetenv("WP_DEVELOPMENT_MODE")
	c := &wp.Context{Config: &config.Config{DevelopmentMode: "all"}}
	for _, mode := range []string{"core", "plugin", "theme", "all"} {
		if !wp.IsDevelopmentMode(c, mode) {
			t.Fatalf("all should cover %q", mode)
		}
	}
	c.Config.DevelopmentMode = "core"
	if !wp.IsDevelopmentMode(c, "core") {
		t.Fatal("expected core match")
	}
	if wp.IsDevelopmentMode(c, "plugin") {
		t.Fatal("core should not cover plugin")
	}
}

func TestGetDevelopmentModeFromProcessEnv(t *testing.T) {
	t.Setenv("WP_DEVELOPMENT_MODE", "theme")
	c := &wp.Context{Config: &config.Config{}} // empty → fall through to getenv
	if got := wp.GetDevelopmentMode(c); got != "theme" {
		t.Fatalf("got %q want theme", got)
	}
	// Config wins when set.
	c.Config.DevelopmentMode = "plugin"
	if got := wp.GetDevelopmentMode(c); got != "plugin" {
		t.Fatalf("config should win, got %q", got)
	}
}

func TestIsDevelopmentModeFilterAndThemeCache(t *testing.T) {
	os.Unsetenv("WP_DEVELOPMENT_MODE")
	c := &wp.Context{
		Config: &config.Config{DevelopmentMode: "theme"},
		Hooks:  wp.NewHooks(),
	}
	if !wp.IsDevelopmentMode(c, "theme") {
		t.Fatal("expected theme development mode")
	}
	if wp.ShouldCacheThemeData(c) {
		t.Fatal("theme development mode should disable theme cache")
	}
	c.Hooks.AddFilter("wp_is_development_mode", func(value any, _ ...any) any {
		return false
	}, 10, 2)
	if wp.IsDevelopmentMode(c, "theme") {
		t.Fatal("filter should force false")
	}
	if !wp.ShouldCacheThemeData(c) {
		t.Fatal("filtered-off theme mode should allow cache")
	}
}

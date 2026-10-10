package wp_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/robertbaehr/GoPress/internal/config"
	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestApplyDebugModeDefaultsOff(t *testing.T) {
	c := &wp.Context{
		Config: &config.Config{Debug: false, DebugDisplay: true, DebugLog: "false", ABSPATH: "/tmp/site/"},
		Hooks:  wp.NewHooks(),
	}
	wp.ApplyDebugMode(c)
	if c.Debug.Enabled || c.Debug.Display || c.Debug.DisplayErrors {
		t.Fatalf("expected debug off: %+v", c.Debug)
	}
	if c.Debug.LogPath != "" {
		t.Fatalf("expected empty log path: %q", c.Debug.LogPath)
	}
	if !c.Debug.Applied {
		t.Fatal("expected Applied")
	}
}

func TestApplyDebugModeEnabledDisplayAndLog(t *testing.T) {
	c := &wp.Context{
		Config: &config.Config{
			Debug: true, DebugDisplay: true, DebugDisplaySet: true,
			DebugLog: "true", ABSPATH: "/tmp/site/",
		},
		Hooks: wp.NewHooks(),
	}
	wp.ApplyDebugMode(c)
	if !c.Debug.Enabled || !c.Debug.Display || !c.Debug.DisplayErrors {
		t.Fatalf("expected debug+display: %+v", c.Debug)
	}
	wantLog := filepath.Join("/tmp/site", "wp-content", "debug.log")
	if c.Debug.LogPath != wantLog {
		t.Fatalf("LogPath=%q want %q", c.Debug.LogPath, wantLog)
	}
}

func TestApplyDebugModeCustomLogPath(t *testing.T) {
	c := &wp.Context{
		Config: &config.Config{
			Debug: true, DebugDisplay: false, DebugDisplaySet: true,
			DebugLog: "/var/log/gopress.log", ABSPATH: "/tmp/site/",
		},
		Hooks: wp.NewHooks(),
	}
	wp.ApplyDebugMode(c)
	if c.Debug.Display || c.Debug.DisplayErrors {
		t.Fatalf("display should be off: %+v", c.Debug)
	}
	if c.Debug.LogPath != "/var/log/gopress.log" {
		t.Fatalf("LogPath=%q", c.Debug.LogPath)
	}
}

func TestApplyDebugModeFilterSkip(t *testing.T) {
	c := &wp.Context{
		Config: &config.Config{Debug: true, DebugDisplay: true, DebugLog: "true"},
		Hooks:  wp.NewHooks(),
	}
	c.Hooks.AddFilter("enable_wp_debug_mode_checks", func(value any, _ ...any) any {
		return false
	}, 10, 0)
	wp.ApplyDebugMode(c)
	if !c.Debug.Skipped || c.Debug.Enabled {
		t.Fatalf("expected skipped: %+v", c.Debug)
	}
	if wp.IsDebugDisplay(c) {
		t.Fatal("IsDebugDisplay should be false when skipped")
	}
}

func TestApplyDebugModeSuppressesRESTDisplayErrors(t *testing.T) {
	c := &wp.Context{
		Config: &config.Config{
			Debug: true, DebugDisplay: true, DebugDisplaySet: true, DebugLog: "false",
		},
		Hooks: wp.NewHooks(),
		Scope: &wp.RequestScope{},
	}
	c.Scope.SetREST(true)
	wp.ApplyDebugMode(c)
	if !c.Debug.Display {
		t.Fatal("constant Display should remain true")
	}
	if c.Debug.DisplayErrors {
		t.Fatal("DisplayErrors should be suppressed for REST")
	}
	if !wp.IsDebugDisplay(c) {
		t.Fatal("IsDebugDisplay uses constants, not DisplayErrors")
	}
}

func TestApplyDebugModeSuppressesJSONRequest(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept", "application/json")
	c := &wp.Context{
		Config: &config.Config{Debug: true, DebugDisplay: true, DebugDisplaySet: true},
		Hooks:  wp.NewHooks(),
		Req:    req,
	}
	wp.ApplyDebugMode(c)
	if c.Debug.DisplayErrors {
		t.Fatal("JSON Accept should suppress DisplayErrors")
	}
}

func TestIsDebugDisplayFromConfigBeforeApply(t *testing.T) {
	c := &wp.Context{
		Config: &config.Config{Debug: true, DebugDisplay: true},
	}
	if !wp.IsDebugDisplay(c) {
		t.Fatal("expected IsDebugDisplay from config")
	}
}

func TestConfigParsesDebugDisplayAndLog(t *testing.T) {
	dir := t.TempDir()
	src := `<?php
define('DB_NAME', 'x');
define('DB_USER', 'x');
define('DB_PASSWORD', 'x');
define('DB_HOST', 'localhost');
define('AUTH_KEY', 'a');
define('SECURE_AUTH_KEY', 'a');
define('LOGGED_IN_KEY', 'a');
define('NONCE_KEY', 'a');
define('AUTH_SALT', 'a');
define('SECURE_AUTH_SALT', 'a');
define('LOGGED_IN_SALT', 'a');
define('NONCE_SALT', 'a');
define('WP_DEBUG', true);
define('WP_DEBUG_DISPLAY', false);
define('WP_DEBUG_LOG', '/tmp/custom-debug.log');
`
	cfgPath := filepath.Join(dir, "wp-config.php")
	if err := os.WriteFile(cfgPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.Load(config.LoadOptions{Root: dir, ConfigPath: cfgPath, SkipEnv: true})
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.Debug {
		t.Fatal("WP_DEBUG")
	}
	if !cfg.DebugDisplaySet || cfg.DebugDisplay {
		t.Fatalf("WP_DEBUG_DISPLAY: set=%v val=%v", cfg.DebugDisplaySet, cfg.DebugDisplay)
	}
	if cfg.DebugLog != "/tmp/custom-debug.log" {
		t.Fatalf("WP_DEBUG_LOG=%q", cfg.DebugLog)
	}
}

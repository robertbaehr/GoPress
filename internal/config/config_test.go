package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/robertbaehr/GoPress/internal/config"
)

func TestLoadOracleTemplate(t *testing.T) {
	root := t.TempDir()
	// go test runs with cwd = package directory.
	src, err := os.ReadFile(filepath.Join("..", "..", "tools", "oracle", "wp-config.php.tmpl"))
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "wp-config.php")
	if err := os.WriteFile(path, src, 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.Load(config.LoadOptions{Root: root, SkipEnv: true})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DBName != "gopress_ref" || cfg.DBUser != "gopress" {
		t.Fatalf("db=%s/%s", cfg.DBName, cfg.DBUser)
	}
	if cfg.TablePrefix != "wp_" {
		t.Fatalf("prefix=%q", cfg.TablePrefix)
	}
	if cfg.AuthKey == "" || cfg.NonceSalt == "" {
		t.Fatal("salts missing")
	}
	if !cfg.DisableWPCron || cfg.Debug {
		t.Fatalf("flags cron=%v debug=%v", cfg.DisableWPCron, cfg.Debug)
	}
	if cfg.Salt("auth") != cfg.AuthKey+cfg.AuthSalt {
		t.Fatal("Salt(auth) mismatch")
	}
}

func TestEnvOverrides(t *testing.T) {
	root := t.TempDir()
	body := `<?php
define('DB_NAME', 'fromfile');
define('DB_USER', 'u');
define('DB_PASSWORD', 'p');
define('DB_HOST', 'localhost');
define('AUTH_KEY', 'a');
define('SECURE_AUTH_KEY', 'b');
define('LOGGED_IN_KEY', 'c');
define('NONCE_KEY', 'd');
define('AUTH_SALT', 'e');
define('SECURE_AUTH_SALT', 'f');
define('LOGGED_IN_SALT', 'g');
define('NONCE_SALT', 'h');
$table_prefix = 'wp_';
`
	if err := os.WriteFile(filepath.Join(root, "wp-config.php"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GOPRESS_DB_NAME", "fromenv")
	cfg, err := config.Load(config.LoadOptions{Root: root})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DBName != "fromenv" {
		t.Fatalf("got %q", cfg.DBName)
	}
}

func TestValidateMissing(t *testing.T) {
	_, err := config.Load(config.LoadOptions{Root: t.TempDir(), SkipEnv: true})
	if err == nil {
		t.Fatal("expected error")
	}
}

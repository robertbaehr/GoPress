package wp_test

import (
	"context"
	"os"
	"testing"

	"github.com/robertbaehr/GoPress/internal/config"
	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestBootLoadsBlogname(t *testing.T) {
	cfg := &config.Config{
		DBName:      "gopress_ref",
		DBUser:      "gopress",
		DBPassword:  "gopress",
		DBHost:      "127.0.0.1:3306",
		DBCharset:   "utf8mb4",
		TablePrefix: "wp_",
		AuthKey:     "k", SecureAuthKey: "k", LoggedInKey: "k", NonceKey: "k",
		AuthSalt: "s", SecureAuthSalt: "s", LoggedInSalt: "s", NonceSalt: "s",
	}
	if dsn := os.Getenv("GOPRESS_TEST_DSN"); dsn != "" {
		// Prefer explicit DSN host/db via Open path: parse is not available,
		// so keep the default gopress_ref credentials when DSN is set only for db tests.
		_ = dsn
	}
	c, err := wp.Boot(context.Background(), cfg)
	if err != nil {
		t.Skipf("Boot unavailable: %v", err)
	}
	defer c.DB.Close()

	if c.Hooks == nil || c.Cache == nil || c.Options == nil || c.Meta == nil || c.Locale == nil || c.Auth == nil || c.Users == nil {
		t.Fatal("Hooks/Cache/Options/Meta/Locale/Auth/Users not wired")
	}
	got := c.Options.GetOption("blogname", "")
	s, _ := got.(string)
	if s == "" {
		t.Fatalf("blogname empty after Boot; got %#v", got)
	}
}

func TestBootNilConfig(t *testing.T) {
	if _, err := wp.Boot(context.Background(), nil); err == nil {
		t.Fatal("expected error")
	}
}

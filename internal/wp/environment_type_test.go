package wp_test

import (
	"os"
	"testing"

	"github.com/robertbaehr/GoPress/internal/config"
	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestGetEnvironmentTypeDefaultProduction(t *testing.T) {
	os.Unsetenv("WP_ENVIRONMENT_TYPE")
	if got := wp.GetEnvironmentType(nil); got != "production" {
		t.Fatalf("nil context: got %q", got)
	}
	c := &wp.Context{}
	if got := wp.GetEnvironmentType(c); got != "production" {
		t.Fatalf("empty context: got %q", got)
	}
	c.Config = &config.Config{}
	if got := wp.GetEnvironmentType(c); got != "production" {
		t.Fatalf("empty config: got %q", got)
	}
}

func TestGetEnvironmentTypeFromConfig(t *testing.T) {
	os.Unsetenv("WP_ENVIRONMENT_TYPE")
	c := &wp.Context{Config: &config.Config{EnvironmentType: "staging"}}
	if got := wp.GetEnvironmentType(c); got != "staging" {
		t.Fatalf("got %q want staging", got)
	}
	c.Config.EnvironmentType = "LOCAL"
	if got := wp.GetEnvironmentType(c); got != "local" {
		t.Fatalf("got %q want local", got)
	}
	c.Config.EnvironmentType = "nope"
	if got := wp.GetEnvironmentType(c); got != "production" {
		t.Fatalf("invalid should fall back to production, got %q", got)
	}
}

func TestGetEnvironmentTypeFromProcessEnv(t *testing.T) {
	t.Setenv("WP_ENVIRONMENT_TYPE", "development")
	c := &wp.Context{Config: &config.Config{}} // empty → fall through to getenv
	if got := wp.GetEnvironmentType(c); got != "development" {
		t.Fatalf("got %q want development", got)
	}
	// Config wins when set (define overrides getenv).
	c.Config.EnvironmentType = "staging"
	if got := wp.GetEnvironmentType(c); got != "staging" {
		t.Fatalf("config should win, got %q", got)
	}
}

func TestGetEnvironmentTypeEmptyEnvIsProduction(t *testing.T) {
	// WP: getenv present but empty → not in allow-list → production.
	t.Setenv("WP_ENVIRONMENT_TYPE", "")
	c := &wp.Context{Config: &config.Config{}}
	if got := wp.GetEnvironmentType(c); got != "production" {
		t.Fatalf("empty getenv → production, got %q", got)
	}
	t.Setenv("WP_ENVIRONMENT_TYPE", "   ")
	if got := wp.GetEnvironmentType(c); got != "production" {
		t.Fatalf("whitespace getenv → production, got %q", got)
	}
}

func TestIsEnvironmentType(t *testing.T) {
	os.Unsetenv("WP_ENVIRONMENT_TYPE")
	c := &wp.Context{Config: &config.Config{EnvironmentType: "local"}}
	if !wp.IsEnvironmentType(c, "local") {
		t.Fatal("expected local match")
	}
	if !wp.IsEnvironmentType(c, "LOCAL") {
		t.Fatal("expected case-insensitive match")
	}
	if wp.IsEnvironmentType(c, "staging") {
		t.Fatal("local should not match staging")
	}
	if wp.IsEnvironmentType(c, "nope") {
		t.Fatal("unknown type never matches")
	}
}

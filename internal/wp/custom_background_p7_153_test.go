package wp_test

import (
	"context"
	"os"
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// P7.153: Get/Set background color and image theme_mods.
func TestCustomBackgroundThemeModsP7153(t *testing.T) {
	if os.Getenv("GOPRESS_SKIP_DB") != "" {
		t.Skip("GOPRESS_SKIP_DB set")
	}
	c, err := wp.Boot(context.Background(), testBootConfig())
	if err != nil {
		t.Skipf("Boot unavailable: %v", err)
	}
	t.Cleanup(func() { _ = c.DB.Close() })
	t.Cleanup(func() {
		_ = wp.SetBackgroundColor(c, "")
		_ = wp.SetBackgroundImage(c, "")
	})

	if err := wp.SetBackgroundColor(c, "#abc"); err != nil {
		t.Fatal(err)
	}
	if got := wp.GetBackgroundColor(c); got != "abc" {
		t.Fatalf("color=%q", got)
	}
	if err := wp.SetBackgroundImage(c, "https://example.com/bg.jpg"); err != nil {
		t.Fatal(err)
	}
	if got := wp.GetBackgroundImage(c); got != "https://example.com/bg.jpg" {
		t.Fatalf("image=%q", got)
	}
	if err := wp.SetBackgroundColor(c, ""); err != nil {
		t.Fatal(err)
	}
	if got := wp.GetBackgroundColor(c); got != "" {
		t.Fatalf("cleared color=%q", got)
	}
}

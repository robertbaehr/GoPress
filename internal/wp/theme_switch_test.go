package wp_test

import (
	"context"
	"os"
	"testing"

	"github.com/robertbaehr/GoPress/internal/theme"
	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestSwitchThemeUpdatesOptions(t *testing.T) {
	if os.Getenv("GOPRESS_SKIP_DB") != "" {
		t.Skip("GOPRESS_SKIP_DB set")
	}
	// Use gopress_go so oracle (gopress_ref) fixture theme stays intact for Boot/diff.
	cfg := testBootConfig()
	cfg.DBName = "gopress_go"
	cfg.ABSPATH = "/workspace/"
	c, err := wp.Boot(context.Background(), cfg)
	if err != nil {
		t.Skipf("Boot unavailable: %v", err)
	}
	t.Cleanup(func() { _ = c.DB.Close() })

	prev := optionString(c.Options.GetOption("stylesheet", ""))
	if prev == "" {
		prev = "twentytwentyfive"
	}
	// Restore before Close (Cleanup LIFO — register restore after Close registration).
	t.Cleanup(func() {
		_ = wp.SwitchTheme(c, "/workspace/", prev)
	})

	themes, err := theme.ListInstalledThemes("/workspace/")
	if err != nil || len(themes) < 2 {
		t.Fatalf("themes: %v len=%d", err, len(themes))
	}
	target := ""
	for _, th := range themes {
		if th.Slug != prev && th.Slug != "" {
			target = th.Slug
			break
		}
	}
	if target == "" {
		t.Fatal("no alternate theme")
	}
	if err := wp.SwitchTheme(c, "/workspace/", target); err != nil {
		t.Fatalf("switch: %v", err)
	}
	if got := optionString(c.Options.GetOption("stylesheet", "")); got != target {
		t.Fatalf("stylesheet=%q want %q", got, target)
	}
	if optionString(c.Options.GetOption("template", "")) == "" {
		t.Fatal("empty template")
	}
}

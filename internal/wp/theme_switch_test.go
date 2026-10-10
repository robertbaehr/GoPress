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
	c, err := wp.Boot(context.Background(), testBootConfig())
	if err != nil {
		t.Skipf("Boot unavailable: %v", err)
	}
	defer c.DB.Close()

	prev := optionString(c.Options.GetOption("stylesheet", ""))
	t.Cleanup(func() {
		_ = wp.SwitchTheme(c, "/workspace/", prev)
	})

	themes, err := theme.ListInstalledThemes("/workspace/")
	if err != nil || len(themes) < 2 {
		t.Fatalf("themes: %v len=%d", err, len(themes))
	}
	target := ""
	for _, th := range themes {
		if th.Slug != prev {
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

package wp_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// P3.136: emoji_url / emoji_ext / emoji_svg_url / emoji_svg_ext filters.
func TestEmojiSettingsFiltersP3136(t *testing.T) {
	c := &wp.Context{Hooks: wp.NewHooks()}
	c.Hooks.AddFilter("emoji_url", func(value any, _ ...any) any {
		return "https://cdn.example/emoji/"
	}, 10, 1)
	c.Hooks.AddFilter("emoji_ext", func(value any, _ ...any) any {
		return ".webp"
	}, 10, 1)
	c.Hooks.AddFilter("emoji_svg_url", func(value any, _ ...any) any {
		return "https://cdn.example/svg/"
	}, 10, 1)
	c.Hooks.AddFilter("emoji_svg_ext", func(value any, _ ...any) any {
		return ".svgz"
	}, 10, 1)

	html := wp.EmojiHeadHTMLCtx(c)
	for _, want := range []string{
		`id="wp-emoji-settings"`,
		`https://cdn.example/emoji/`,
		`.webp`,
		`https://cdn.example/svg/`,
		`.svgz`,
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("missing %q in %s", want, html)
		}
	}
	// Loader still skipped.
	if wp.PrintEmojiDetectionScript() != "" {
		t.Fatal("loader should stay empty")
	}
}

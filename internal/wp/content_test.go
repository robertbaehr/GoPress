package wp_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestTheContentPipelineBlocksShortcodesTexturize(t *testing.T) {
	c := &wp.Context{
		Hooks:      wp.NewHooks(),
		Shortcodes: wp.NewShortcodes(),
	}
	wp.RegisterDefaultShortcodes(c.Shortcodes)
	c.Shortcodes.AddShortcode("hello", func(attrs map[string]string, _ string) string {
		return "Hi " + attrs["name"]
	})

	in := `<!-- wp:paragraph --><p>Say [hello name="Bob"] and...</p><!-- /wp:paragraph -->`
	out := wp.TheContent(c, in)
	if !strings.Contains(out, "Hi Bob") {
		t.Fatalf("shortcode missing: %q", out)
	}
	if !strings.Contains(out, "&#8230;") {
		t.Fatalf("texturize missing: %q", out)
	}
	if strings.Contains(out, "<!-- wp:") {
		t.Fatalf("blocks not rendered: %q", out)
	}
}

func TestTheContentFilterHook(t *testing.T) {
	c := &wp.Context{Hooks: wp.NewHooks(), Shortcodes: wp.NewShortcodes()}
	c.Hooks.AddFilter("the_content", func(value any, _ ...any) any {
		return value.(string) + "<!--filtered-->"
	}, 10, 1)
	out := wp.TheContent(c, "<p>plain</p>")
	if !strings.HasSuffix(out, "<!--filtered-->") {
		t.Fatalf("got %q", out)
	}
}

func TestTexturizeApostrophe(t *testing.T) {
	got := wp.Texturize("it's")
	if got != "it&#8217;s" {
		t.Fatalf("got %q", got)
	}
}

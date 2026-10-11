package blocks_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/blocks"
)

// P5.140: core/read-more honors supports.anchor and typography.fontSize.
func TestRenderReadMoreAnchorFontSizeP5140(t *testing.T) {
	env := &blocks.Env{}
	env.SetCurrentPost(&blocks.QueryPost{ID: 7, Title: "Hello", Link: "/?p=7"})
	out := blocks.RenderWith(env, `<!-- wp:read-more {"content":"Keep reading","fontSize":"large","anchor":"read-more-hello","className":"extra-rm","style":{"elements":{"link":{"color":{"text":"var:preset|color|accent"}}}}} /-->`)
	for _, want := range []string{
		`wp-block-read-more`,
		`has-large-font-size`,
		`id="read-more-hello"`,
		`extra-rm`,
		`has-link-color`,
		`Keep reading`,
		`href="/?p=7"`,
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in: %q", want, out)
		}
	}
}

func TestRenderReadMoreEmptyWithoutPostP5140(t *testing.T) {
	out := blocks.RenderWith(&blocks.Env{}, `<!-- wp:read-more {"anchor":"should-not-appear","fontSize":"small"} /-->`)
	if strings.TrimSpace(out) != "" {
		t.Fatalf("expected empty without post, got %q", out)
	}
}

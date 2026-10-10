package blocks_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/blocks"
)

func TestRenderReadMorePolishClassNameAndLinkColor(t *testing.T) {
	env := &blocks.Env{}
	env.SetCurrentPost(&blocks.QueryPost{
		ID:    7,
		Title: "Polish Me",
		Link:  "http://example.test/polish-me/",
	})
	out := blocks.RenderWith(env, `<!-- wp:read-more {"content":"Keep going","className":"extra-read","style":{"elements":{"link":{"color":{"text":"var:preset|color|accent"}}}}} /-->`)
	for _, want := range []string{
		`wp-block-read-more`,
		`has-link-color`,
		`extra-read`,
		`href="http://example.test/polish-me/"`,
		`Keep going`,
		`: Polish Me`,
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in %q", want, out)
		}
	}
}

func TestRenderReadMoreFallbackPermalink(t *testing.T) {
	env := &blocks.Env{}
	env.SetCurrentPost(&blocks.QueryPost{ID: 99, Title: "No Link"})
	out := blocks.RenderWith(env, `<!-- wp:read-more /-->`)
	if !strings.Contains(out, `href="/?p=99"`) {
		t.Fatalf("expected /?p= fallback: %q", out)
	}
	if !strings.Contains(out, "Read more") {
		t.Fatalf("expected default text: %q", out)
	}
}

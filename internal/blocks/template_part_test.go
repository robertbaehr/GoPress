package blocks_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/blocks"
)

func TestRenderTemplatePart(t *testing.T) {
	env := &blocks.Env{
		LoadTemplatePart: func(slug string) (string, bool) {
			if slug != "header" {
				return "", false
			}
			return `<!-- wp:paragraph --><p>Customized header part</p><!-- /wp:paragraph -->`, true
		},
	}
	in := `<!-- wp:template-part {"slug":"header","tagName":"header"} /-->`
	out := blocks.RenderWith(env, in)
	if !strings.Contains(out, `<header class="wp-block-template-part">`) {
		t.Fatalf("missing wrapper: %q", out)
	}
	if !strings.Contains(out, `<p>Customized header part</p>`) {
		t.Fatalf("missing inner: %q", out)
	}
	if strings.Contains(out, "<!-- wp:") {
		t.Fatalf("leftover comments: %q", out)
	}
}

func TestRenderTemplatePartMissing(t *testing.T) {
	env := &blocks.Env{
		LoadTemplatePart: func(slug string) (string, bool) { return "", false },
	}
	out := blocks.RenderWith(env, `<!-- wp:template-part {"slug":"nope"} /-->`)
	if out != "" {
		t.Fatalf("want empty, got %q", out)
	}
}

func TestRenderTemplatePartCycleGuard(t *testing.T) {
	env := &blocks.Env{}
	env.LoadTemplatePart = func(slug string) (string, bool) {
		// Part that references itself.
		return `<!-- wp:template-part {"slug":"loop"} /--><!-- wp:paragraph --><p>X</p><!-- /wp:paragraph -->`, true
	}
	out := blocks.RenderWith(env, `<!-- wp:template-part {"slug":"loop","tagName":"div"} /-->`)
	if strings.Count(out, `<p>X</p>`) != 1 {
		t.Fatalf("cycle not guarded: %q", out)
	}
}

package blocks_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/blocks"
)

func TestTermDescriptionBlock(t *testing.T) {
	env := &blocks.Env{
		TermContext: &blocks.TermBindingContext{
			ID:          1,
			Taxonomy:    "category",
			Name:        "Uncategorized",
			Description: "<p>Category blurb.</p>",
		},
	}
	out := blocks.RenderWith(env, `<!-- wp:term-description {"textAlign":"center","align":"wide"} /-->`)
	for _, want := range []string{
		`class="wp-block-term-description has-text-align-center alignwide"`,
		`<p>Category blurb.</p>`,
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in %q", want, out)
		}
	}
}

func TestTermDescriptionEmptyWithoutContext(t *testing.T) {
	out := blocks.RenderWith(&blocks.Env{}, `<!-- wp:term-description /-->`)
	if strings.TrimSpace(out) != "" {
		t.Fatalf("expected empty without TermContext, got %q", out)
	}
}

func TestTermDescriptionLinkColorClass(t *testing.T) {
	env := &blocks.Env{
		TermContext: &blocks.TermBindingContext{
			ID:          2,
			Taxonomy:    "post_tag",
			Description: "Tag note",
		},
	}
	markup := `<!-- wp:term-description {"style":{"elements":{"link":{"color":{"text":"var:preset|color|accent-1"}}}}} /-->`
	out := blocks.RenderWith(env, markup)
	if !strings.Contains(out, "has-link-color") {
		t.Fatalf("expected has-link-color: %q", out)
	}
	if !strings.Contains(out, "Tag note") {
		t.Fatalf("expected description: %q", out)
	}
}

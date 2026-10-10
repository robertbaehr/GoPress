package blocks_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/blocks"
)

func TestRenderQuoteSynthesizeValueCitationAlign(t *testing.T) {
	out := blocks.Render(`<!-- wp:quote {"textAlign":"center","align":"wide","className":"is-style-plain","value":"<p>Hello quote</p>","citation":"Anon"} /-->`)
	for _, want := range []string{
		`<blockquote class="`,
		`wp-block-quote`,
		`has-text-align-center`,
		`alignwide`,
		`is-style-plain`,
		`is-layout-flow`,
		`wp-block-quote-is-layout-flow`,
		`<p>Hello quote</p>`,
		`<cite>Anon</cite>`,
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in %q", want, out)
		}
	}
}

func TestRenderQuotePrefersSavedMarkup(t *testing.T) {
	src := `<!-- wp:quote -->
<blockquote class="wp-block-quote"><p>Saved</p><cite>Cite</cite></blockquote>
<!-- /wp:quote -->`
	out := blocks.Render(src)
	if !strings.Contains(out, `<p>Saved</p>`) || !strings.Contains(out, `<cite>Cite</cite>`) {
		t.Fatalf("lost saved markup: %q", out)
	}
	if !strings.Contains(out, `is-layout-flow`) || !strings.Contains(out, `wp-block-quote-is-layout-flow`) {
		t.Fatalf("missing layout classes: %q", out)
	}
	if strings.Count(out, `wp-block-quote`) < 1 {
		t.Fatalf("missing quote class: %q", out)
	}
}

func TestRenderPullquoteSynthesize(t *testing.T) {
	out := blocks.Render(`<!-- wp:pullquote {"align":"left","className":"is-style-solid-color","value":"Pull me","citation":"Author"} /-->`)
	for _, want := range []string{
		`<figure class="`,
		`wp-block-pullquote`,
		`alignleft`,
		`is-style-solid-color`,
		`<p>Pull me</p>`,
		`<cite>Author</cite>`,
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in %q", want, out)
		}
	}
}

func TestRenderPullquotePrefersSavedMarkup(t *testing.T) {
	src := `<!-- wp:pullquote -->
<figure class="wp-block-pullquote"><blockquote><p>Saved pull</p><cite>Cap</cite></blockquote></figure>
<!-- /wp:pullquote -->`
	out := blocks.Render(src)
	if !strings.Contains(out, `Saved pull`) || !strings.Contains(out, `<cite>Cap</cite>`) {
		t.Fatalf("lost saved pullquote: %q", out)
	}
	if strings.Count(out, `wp-block-pullquote`) != 1 {
		t.Fatalf("expected single pullquote: %q", out)
	}
}

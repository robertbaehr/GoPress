package blocks_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/blocks"
)

func TestRenderParagraphHeading(t *testing.T) {
	in := `<!-- wp:heading -->
<h2 class="wp-block-heading">Hello</h2>
<!-- /wp:heading -->
<!-- wp:paragraph -->
<p>World</p>
<!-- /wp:paragraph -->`
	out := blocks.Render(in)
	if !strings.Contains(out, `<h2 class="wp-block-heading">Hello</h2>`) {
		t.Fatalf("missing heading: %q", out)
	}
	if !strings.Contains(out, `<p>World</p>`) {
		t.Fatalf("missing paragraph: %q", out)
	}
	if strings.Contains(out, "<!-- wp:") {
		t.Fatalf("comments should be stripped: %q", out)
	}
}

func TestRenderListNested(t *testing.T) {
	doc := readSerialized(t, "core__list__ul")
	out := blocks.Render(doc)
	if !strings.Contains(out, `<ul class="wp-block-list">`) {
		t.Fatalf("missing ul: %q", out)
	}
	if !strings.Contains(out, `<li>Text &amp; Headings</li>`) {
		t.Fatalf("missing list item: %q", out)
	}
	if strings.Contains(out, "<!-- wp:") {
		t.Fatalf("leftover comments: %q", out)
	}
}

func TestRenderQuoteNested(t *testing.T) {
	doc := readSerialized(t, "core__quote__style-1")
	out := blocks.Render(doc)
	if !strings.Contains(out, `<blockquote class="wp-block-quote">`) {
		t.Fatalf("missing quote: %q", out)
	}
	if !strings.Contains(out, `<p>The editor will endeavour`) {
		t.Fatalf("missing inner paragraph: %q", out)
	}
	if !strings.Contains(out, `<cite>Matt Mullenweg, 2017</cite>`) {
		t.Fatalf("missing cite: %q", out)
	}
}

func TestRenderCodeHTMLSeparatorImage(t *testing.T) {
	cases := []struct {
		fixture string
		needle  string
	}{
		{"core__code", `<pre class="wp-block-code">`},
		{"core__html", `<h1>Some HTML code</h1>`},
		{"core__separator", `<hr class="wp-block-separator`},
		{"core__image", `<figure class="wp-block-image">`},
	}
	for _, tc := range cases {
		out := blocks.Render(readSerialized(t, tc.fixture))
		if !strings.Contains(out, tc.needle) {
			t.Fatalf("%s: missing %q in %q", tc.fixture, tc.needle, out)
		}
	}
}

func TestRenderUnknownUsesInnerHTML(t *testing.T) {
	in := `<!-- wp:custom/widget -->
<div class="widget">Hi</div>
<!-- /wp:custom/widget -->`
	out := blocks.Render(in)
	if out != "\n<div class=\"widget\">Hi</div>\n" {
		t.Fatalf("got %q", out)
	}
}

func TestRenderVoidSeparator(t *testing.T) {
	out := blocks.Render(`<!-- wp:separator {"className":"is-style-wide"} /-->`)
	if !strings.Contains(out, `class="wp-block-separator is-style-wide"`) {
		t.Fatalf("got %q", out)
	}
}

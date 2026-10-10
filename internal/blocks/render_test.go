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
	if !strings.Contains(out, `<p class="wp-block-paragraph">World</p>`) {
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
	if !strings.Contains(out, `class="wp-block-quote`) {
		t.Fatalf("missing quote: %q", out)
	}
	if !strings.Contains(out, `<p class="wp-block-paragraph">The editor will endeavour`) {
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

func TestRenderMoreCoreBlocks(t *testing.T) {
	cases := []struct {
		fixture string
		needles []string
	}{
		{"core__group", []string{`class="wp-block-group`, `<p class="wp-block-paragraph">This is a group block.</p>`, `<p class="wp-block-paragraph">Group block content.</p>`}},
		{"core__columns", []string{`class="wp-block-columns`, `class="wp-block-column`, `Column One, Paragraph One`}},
		{"core__column", []string{`class="wp-block-column`}},
		{"core__buttons", []string{`class="wp-block-buttons`, `wp-block-button__link`, `My button 1`}},
		{"core__button__squared", []string{`class="wp-block-button`, `My button`}},
		{"core__spacer", []string{`class="wp-block-spacer`, `height:100px`}},
		{"core__pullquote", []string{`class="wp-block-pullquote`, `Testing pullquote block`}},
		{"core__preformatted", []string{`class="wp-block-preformatted`, `preformatted`}},
		{"core__verse", []string{`class="wp-block-verse`, `verse`}},
		{"core__table", []string{`class="wp-block-table`, `<table`, `Musician`}},
		{"core__embed", []string{`class="wp-block-embed`, `https://example.com/`}},
		{"core__file__new-window", []string{`class="wp-block-file`, `Download`}},
		{"core__audio", []string{`class="wp-block-audio`, `<audio`}},
		{"core__video", []string{`class="wp-block-video`, `<video`}},
		{"core__cover", []string{`class="wp-block-cover`, `Guten Berg!`}},
		{"core__media-text", []string{`class="wp-block-media-text`, `My Content`}},
	}
	for _, tc := range cases {
		out := blocks.Render(readSerialized(t, tc.fixture))
		for _, needle := range tc.needles {
			if !strings.Contains(out, needle) {
				t.Fatalf("%s: missing %q in %q", tc.fixture, needle, truncate(out, 240))
			}
		}
		if strings.Contains(out, "<!-- wp:") {
			t.Fatalf("%s: leftover comments: %q", tc.fixture, truncate(out, 240))
		}
	}
}

func TestRenderVoidSpacer(t *testing.T) {
	out := blocks.Render(`<!-- wp:spacer {"height":"48px"} /-->`)
	if !strings.Contains(out, `style="height:48px"`) || !strings.Contains(out, `class="wp-block-spacer"`) {
		t.Fatalf("got %q", out)
	}
}

func TestRenderEmbedLinkFallback(t *testing.T) {
	out := blocks.Render(`<!-- wp:embed {"url":"https://example.org/watch","caption":"Demo"} /-->`)
	if !strings.Contains(out, `<a href="https://example.org/watch">https://example.org/watch</a>`) {
		t.Fatalf("missing link fallback: %q", out)
	}
	if !strings.Contains(out, `Demo`) {
		t.Fatalf("missing caption: %q", out)
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

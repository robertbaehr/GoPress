package blocks_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/blocks"
)

func TestRenderListAddsWpBlockListClass(t *testing.T) {
	src := `<!-- wp:list -->
<ul><li>One</li></ul>
<!-- /wp:list -->`
	out := blocks.Render(src)
	if !strings.Contains(out, `class="wp-block-list"`) && !strings.Contains(out, `wp-block-list`) {
		t.Fatalf("expected wp-block-list class: %q", out)
	}
	if !strings.Contains(out, "<li>One</li>") {
		t.Fatalf("missing item: %q", out)
	}
}

func TestRenderListSynthesizeOrderedAttrs(t *testing.T) {
	out := blocks.Render(`<!-- wp:list {"ordered":true,"start":3,"reversed":true,"type":"a","values":"<li>Alpha</li><li>Beta</li>","className":"is-style-plain"} /-->`)
	for _, want := range []string{
		`<ol class="`,
		`wp-block-list`,
		`is-style-plain`,
		`start="3"`,
		` reversed`,
		`type="a"`,
		`<li>Alpha</li>`,
		`<li>Beta</li>`,
		`</ol>`,
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in %q", want, out)
		}
	}
}

func TestRenderListSynthesizeUnordered(t *testing.T) {
	out := blocks.Render(`<!-- wp:list {"values":"<li>A</li><li>B</li>"} /-->`)
	if !strings.Contains(out, `<ul class="wp-block-list">`) {
		t.Fatalf("expected ul: %q", out)
	}
	if strings.Contains(out, "<ol") {
		t.Fatalf("unexpected ol: %q", out)
	}
}

func TestRenderListItemSynthesizeContent(t *testing.T) {
	out := blocks.Render(`<!-- wp:list-item {"content":"Hello <em>item</em>"} /-->`)
	if out != "<li>Hello <em>item</em></li>" {
		t.Fatalf("got %q", out)
	}
}

func TestRenderListItemPrefersSavedMarkup(t *testing.T) {
	src := `<!-- wp:list-item -->
<li>Saved item</li>
<!-- /wp:list-item -->`
	out := blocks.Render(src)
	if !strings.Contains(out, "<li>Saved item</li>") {
		t.Fatalf("lost saved markup: %q", out)
	}
}

func TestRenderListNestedFixtureKeepsClass(t *testing.T) {
	doc := readSerialized(t, "core__list__ul")
	out := blocks.Render(doc)
	if !strings.Contains(out, `<ul class="wp-block-list">`) {
		t.Fatalf("missing ul: %q", out)
	}
}

func TestRenderListPolishAnchorClassNameColors(t *testing.T) {
	// P5.104: polish saved markup + applySupports colors.
	src := `<!-- wp:list {"anchor":"toc","className":"is-style-plain","align":"wide","textColor":"contrast","backgroundColor":"base"} -->
<ul><li>One</li></ul>
<!-- /wp:list -->`
	out := blocks.Render(src)
	for _, want := range []string{
		`wp-block-list`,
		`is-style-plain`,
		`alignwide`,
		`id="toc"`,
		`has-contrast-color`,
		`has-base-background-color`,
		`<li>One</li>`,
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in %q", want, out)
		}
	}
}

func TestRenderListSynthesizeAnchor(t *testing.T) {
	out := blocks.Render(`<!-- wp:list {"ordered":true,"start":2,"anchor":"steps","values":"<li>First</li>"} /-->`)
	if !strings.Contains(out, `id="steps"`) || !strings.Contains(out, `start="2"`) || !strings.Contains(out, `<li>First</li>`) {
		t.Fatalf("expected ordered list with anchor: %q", out)
	}
}

func TestRenderListItemPolishAnchorColors(t *testing.T) {
	src := `<!-- wp:list-item {"anchor":"item-1","textColor":"contrast"} -->
<li>Saved item</li>
<!-- /wp:list-item -->`
	out := blocks.Render(src)
	for _, want := range []string{
		`id="item-1"`,
		`has-contrast-color`,
		`Saved item`,
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in %q", want, out)
		}
	}
}

func TestRenderListItemSynthesizeAnchor(t *testing.T) {
	out := blocks.Render(`<!-- wp:list-item {"content":"Hello","anchor":"li-a"} /-->`)
	if out != `<li id="li-a">Hello</li>` {
		t.Fatalf("got %q", out)
	}
}

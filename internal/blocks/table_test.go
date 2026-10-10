package blocks_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/blocks"
)

func TestRenderTablePolishFixedLayoutAlign(t *testing.T) {
	markup := `<!-- wp:table {"hasFixedLayout":true,"align":"wide","className":"is-style-stripes"} -->
<figure class="wp-block-table"><table><tbody><tr><td>A</td><td>B</td></tr></tbody></table></figure>
<!-- /wp:table -->`
	out := blocks.Render(markup)
	for _, want := range []string{
		`wp-block-table`,
		`alignwide`,
		`is-style-stripes`,
		`has-fixed-layout`,
		`<td>A</td>`,
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in %q", want, out)
		}
	}
}

func TestRenderTableSynthesizeFromAttrs(t *testing.T) {
	out := blocks.Render(`<!-- wp:table {"hasFixedLayout":true,"caption":"Demo","head":[{"cells":[{"content":"H1","tag":"th"},{"content":"H2","tag":"th"}]}],"body":[{"cells":[{"content":"A","tag":"td"},{"content":"B","tag":"td"}]}]} /-->`)
	for _, want := range []string{
		`<figure class="wp-block-table">`,
		`has-fixed-layout`,
		`<thead>`,
		`<th>H1</th>`,
		`<tbody>`,
		`<td>A</td>`,
		`<figcaption class="wp-element-caption">Demo</figcaption>`,
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in %q", want, out)
		}
	}
}

func TestRenderTableHasFixedLayoutFalse(t *testing.T) {
	markup := `<!-- wp:table {"hasFixedLayout":false} -->
<figure class="wp-block-table"><table><tbody><tr><td>X</td></tr></tbody></table></figure>
<!-- /wp:table -->`
	out := blocks.Render(markup)
	if strings.Contains(out, "has-fixed-layout") {
		t.Fatalf("hasFixedLayout=false should omit class: %q", out)
	}
	if !strings.Contains(out, "wp-block-table") {
		t.Fatalf("expected wp-block-table: %q", out)
	}
}

func TestRenderTableFixtureMusician(t *testing.T) {
	out := blocks.Render(readSerialized(t, "core__table"))
	for _, want := range []string{`class="wp-block-table`, `<table`, `Musician`, `has-fixed-layout`} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in %q", want, truncate(out, 240))
		}
	}
}

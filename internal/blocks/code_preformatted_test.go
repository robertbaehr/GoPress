package blocks_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/blocks"
)

func TestRenderCodeSynthesizeContentAlign(t *testing.T) {
	out := blocks.Render(`<!-- wp:code {"align":"wide","className":"is-style-default","content":"echo hello;"} /-->`)
	for _, want := range []string{
		`<pre class="`,
		`wp-block-code`,
		`alignwide`,
		`is-style-default`,
		`<code>echo hello;</code>`,
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in %q", want, out)
		}
	}
}

func TestRenderCodePrefersSavedMarkup(t *testing.T) {
	src := `<!-- wp:code -->
<pre class="wp-block-code"><code>saved();</code></pre>
<!-- /wp:code -->`
	out := blocks.Render(src)
	if !strings.Contains(out, `saved();`) {
		t.Fatalf("lost saved code: %q", out)
	}
	if strings.Count(out, `wp-block-code`) != 1 {
		t.Fatalf("expected single code class: %q", out)
	}
}

func TestRenderPreformattedSynthesize(t *testing.T) {
	out := blocks.Render(`<!-- wp:preformatted {"className":"custom-pre","content":"line1\nline2"} /-->`)
	for _, want := range []string{
		`<pre class="`,
		`wp-block-preformatted`,
		`custom-pre`,
		`line1`,
		`line2`,
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in %q", want, out)
		}
	}
}

func TestRenderPreformattedPrefersSavedMarkup(t *testing.T) {
	src := `<!-- wp:preformatted -->
<pre class="wp-block-preformatted">kept</pre>
<!-- /wp:preformatted -->`
	out := blocks.Render(src)
	if !strings.Contains(out, `kept`) || !strings.Contains(out, `wp-block-preformatted`) {
		t.Fatalf("lost saved preformatted: %q", out)
	}
}

func TestRenderPreformattedPolishAnchorClassNameColors(t *testing.T) {
	// P5.101: polish saved markup + applySupports colors.
	src := `<!-- wp:preformatted {"anchor":"pre-1","className":"is-style-wide","textColor":"contrast","backgroundColor":"base"} -->
<pre class="wp-block-preformatted">kept</pre>
<!-- /wp:preformatted -->`
	out := blocks.Render(src)
	for _, want := range []string{
		`wp-block-preformatted`,
		`is-style-wide`,
		`id="pre-1"`,
		`has-contrast-color`,
		`has-base-background-color`,
		`kept`,
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in %q", want, out)
		}
	}
}

func TestRenderPreformattedSynthesizeAnchor(t *testing.T) {
	out := blocks.Render(`<!-- wp:preformatted {"anchor":"snippet","content":"tab\there"} /-->`)
	if !strings.Contains(out, `id="snippet"`) || !strings.Contains(out, `tab`) {
		t.Fatalf("expected anchor synthesize: %q", out)
	}
}

func TestRenderVerseSynthesizeTextAlign(t *testing.T) {
	out := blocks.Render(`<!-- wp:verse {"textAlign":"center","className":"poem","content":"Roses are red"} /-->`)
	for _, want := range []string{
		`<pre class="`,
		`wp-block-verse`,
		`has-text-align-center`,
		`poem`,
		`Roses are red`,
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in %q", want, out)
		}
	}
}

func TestRenderVersePrefersSavedMarkup(t *testing.T) {
	src := `<!-- wp:verse -->
<pre class="wp-block-verse">stanza</pre>
<!-- /wp:verse -->`
	out := blocks.Render(src)
	if !strings.Contains(out, `stanza`) || !strings.Contains(out, `wp-block-verse`) {
		t.Fatalf("lost saved verse: %q", out)
	}
}

func TestRenderVersePolishAnchorTextAlignColors(t *testing.T) {
	// P5.100: polish saved markup + applySupports colors.
	src := `<!-- wp:verse {"textAlign":"right","anchor":"ode-1","className":"is-style-wide","textColor":"contrast","backgroundColor":"base"} -->
<pre class="wp-block-verse">stanza</pre>
<!-- /wp:verse -->`
	out := blocks.Render(src)
	for _, want := range []string{
		`wp-block-verse`,
		`has-text-align-right`,
		`is-style-wide`,
		`id="ode-1"`,
		`has-contrast-color`,
		`has-base-background-color`,
		`stanza`,
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in %q", want, out)
		}
	}
}

func TestRenderVerseSynthesizeAnchor(t *testing.T) {
	out := blocks.Render(`<!-- wp:verse {"anchor":"lyric","content":"la la"} /-->`)
	if !strings.Contains(out, `id="lyric"`) || !strings.Contains(out, `la la`) {
		t.Fatalf("expected anchor synthesize: %q", out)
	}
}

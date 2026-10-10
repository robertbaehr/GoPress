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

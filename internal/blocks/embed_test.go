package blocks_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/blocks"
)

func TestRenderEmbedLinkFallback(t *testing.T) {
	out := blocks.Render(`<!-- wp:embed {"url":"https://example.org/watch","caption":"Demo"} /-->`)
	if !strings.Contains(out, `<a href="https://example.org/watch">https://example.org/watch</a>`) {
		t.Fatalf("missing link fallback: %q", out)
	}
	if !strings.Contains(out, `Demo`) {
		t.Fatalf("missing caption: %q", out)
	}
	if !strings.Contains(out, `class="wp-block-embed"`) && !strings.Contains(out, `wp-block-embed`) {
		t.Fatalf("missing embed class: %q", out)
	}
}

func TestRenderEmbedProviderClasses(t *testing.T) {
	out := blocks.Render(`<!-- wp:embed {"url":"https://www.youtube.com/watch?v=dQw4w9WgXcQ","providerNameSlug":"youtube","type":"video","responsive":true,"align":"wide"} /-->`)
	for _, needle := range []string{
		`wp-block-embed`,
		`is-type-video`,
		`is-provider-youtube`,
		`wp-block-embed-youtube`,
		`alignwide`,
		`wp-embed-responsive`,
		`https://www.youtube.com/watch?v=dQw4w9WgXcQ`,
	} {
		if !strings.Contains(out, needle) {
			t.Fatalf("missing %q in %q", needle, out)
		}
	}
}

func TestRenderEmbedInfersProviderFromURL(t *testing.T) {
	out := blocks.Render(`<!-- wp:embed {"url":"https://vimeo.com/123456"} /-->`)
	if !strings.Contains(out, `is-provider-vimeo`) || !strings.Contains(out, `wp-block-embed-vimeo`) {
		t.Fatalf("expected inferred vimeo classes: %q", out)
	}
	if !strings.Contains(out, `is-type-video`) {
		t.Fatalf("expected video type for vimeo: %q", out)
	}
}

func TestRenderEmbedSavedHTMLKeepsProviderClasses(t *testing.T) {
	src := `<!-- wp:embed {"url":"https://twitter.com/WordPress/status/1","providerNameSlug":"twitter","type":"rich"} -->
<figure class="wp-block-embed"><div class="wp-block-embed__wrapper">
https://twitter.com/WordPress/status/1
</div></figure>
<!-- /wp:embed -->`
	out := blocks.Render(src)
	if !strings.Contains(out, `is-provider-twitter`) || !strings.Contains(out, `wp-block-embed-twitter`) {
		t.Fatalf("expected twitter classes on saved markup: %q", out)
	}
	if !strings.Contains(out, `is-type-rich`) {
		t.Fatalf("expected rich type: %q", out)
	}
}

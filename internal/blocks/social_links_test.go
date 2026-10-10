package blocks_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/blocks"
)

func TestRenderSocialLinkSelfClosing(t *testing.T) {
	out := blocks.Render(`<!-- wp:social-link {"url":"https://facebook.com/WordPress","service":"facebook"} /-->`)
	for _, needle := range []string{
		`<li class="`,
		`wp-social-link`,
		`wp-social-link-facebook`,
		`href="https://facebook.com/WordPress"`,
		`wp-block-social-link-anchor`,
		`screen-reader-text`,
		`Facebook`,
		`<svg`,
	} {
		if !strings.Contains(out, needle) {
			t.Fatalf("missing %q in %q", needle, out)
		}
	}
}

func TestRenderSocialLinksWithChildren(t *testing.T) {
	src := `<!-- wp:social-links {"openInNewTab":true,"showLabels":true,"className":"is-style-logos-only"} -->
<ul class="wp-block-social-links is-style-logos-only"><!-- wp:social-link {"url":"wordpress.org","service":"wordpress"} /-->
<!-- wp:social-link {"url":"https://x.com/WordPress","service":"x"} /--></ul>
<!-- /wp:social-links -->`
	out := blocks.Render(src)
	for _, needle := range []string{
		`wp-block-social-links`,
		`is-style-logos-only`,
		`wp-social-link-wordpress`,
		`href="https://wordpress.org"`,
		`target="_blank"`,
		`noopener`,
		`wp-social-link-x`,
		`WordPress`,
	} {
		if !strings.Contains(out, needle) {
			t.Fatalf("missing %q in %q", needle, out)
		}
	}
	// showLabels=true → label not screen-reader-only.
	if strings.Count(out, "screen-reader-text") != 0 {
		t.Fatalf("expected visible labels, got: %q", out)
	}
}

func TestRenderSocialLinkEmptyURL(t *testing.T) {
	out := blocks.Render(`<!-- wp:social-link {"service":"facebook"} /-->`)
	if strings.TrimSpace(out) != "" {
		t.Fatalf("expected empty render without url, got %q", out)
	}
}

func TestRenderSocialLinksSynthesizeWrapper(t *testing.T) {
	// No saved InnerHTML wrapper — synthesize <ul> from inner blocks.
	src := `<!-- wp:social-links {"iconColorValue":"#111","align":"center"} -->
<!-- wp:social-link {"url":"#","service":"instagram"} /-->
<!-- /wp:social-links -->`
	out := blocks.Render(src)
	if !strings.Contains(out, `<ul class="`) || !strings.Contains(out, `wp-block-social-links`) {
		t.Fatalf("expected synthesized ul: %q", out)
	}
	if !strings.Contains(out, `aligncenter`) || !strings.Contains(out, `has-icon-color`) {
		t.Fatalf("expected align/icon classes: %q", out)
	}
	if !strings.Contains(out, `wp-social-link-instagram`) {
		t.Fatalf("expected instagram child: %q", out)
	}
}

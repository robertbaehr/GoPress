package blocks_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/blocks"
)

func TestRenderSiteTitlePolishClasses(t *testing.T) {
	env := &blocks.Env{
		SiteName:    func() string { return "GoPress Fixture" },
		HomeURL:     func() string { return "http://127.0.0.1:8081" },
		IsFrontPage: func() bool { return true },
		IsPaged:     func() bool { return false },
	}
	out := blocks.RenderWith(env,
		`<!-- wp:site-title {"textAlign":"center","className":"site-brand","style":{"elements":{"link":{"color":{"text":"var:preset|color|vivid-red"}}}}} /-->`)
	wantClass := `wp-block-site-title has-text-align-center has-link-color site-brand`
	if !strings.Contains(out, wantClass) {
		t.Fatalf("missing classes %q in %q", wantClass, out)
	}
	if !strings.Contains(out, `aria-current="page"`) {
		t.Fatalf("expected aria-current on front: %q", out)
	}
}

func TestRenderSiteTitleNoAriaWhenPaged(t *testing.T) {
	env := &blocks.Env{
		SiteName:    func() string { return "GoPress Fixture" },
		HomeURL:     func() string { return "http://127.0.0.1:8081" },
		IsFrontPage: func() bool { return true },
		IsPaged:     func() bool { return true },
	}
	out := blocks.RenderWith(env, `<!-- wp:site-title /-->`)
	if strings.Contains(out, `aria-current`) {
		t.Fatalf("paged should omit aria-current: %q", out)
	}
}

func TestRenderSiteTaglineClassName(t *testing.T) {
	env := &blocks.Env{
		SiteTagline: func() string { return "Just another site" },
	}
	out := blocks.RenderWith(env,
		`<!-- wp:site-tagline {"textAlign":"right","className":"tagline","level":2} /-->`)
	want := `<h2 class="wp-block-site-tagline has-text-align-right tagline">Just another site</h2>`
	if !strings.Contains(out, want) {
		t.Fatalf("got %q want substring %q", out, want)
	}
}

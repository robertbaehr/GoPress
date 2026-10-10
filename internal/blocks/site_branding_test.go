package blocks_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/blocks"
)

func TestRenderSiteTitleDefaultLink(t *testing.T) {
	env := &blocks.Env{
		SiteName: func() string { return "GoPress Fixture" },
		HomeURL:  func() string { return "http://127.0.0.1:8081" },
	}
	out := blocks.RenderWith(env, `<!-- wp:site-title /-->`)
	if !strings.Contains(out, `class="wp-block-site-title"`) {
		t.Fatalf("missing class: %q", out)
	}
	if !strings.Contains(out, `<a href="http://127.0.0.1:8081" target="_self" rel="home">GoPress Fixture</a>`) {
		t.Fatalf("expected home link, got %q", out)
	}
	if !strings.HasPrefix(strings.TrimSpace(out), "<h1 ") {
		t.Fatalf("expected h1 default, got %q", out)
	}
}

func TestRenderSiteTitleNoLink(t *testing.T) {
	env := &blocks.Env{
		SiteName: func() string { return "GoPress Fixture" },
		HomeURL:  func() string { return "http://127.0.0.1:8081" },
	}
	out := blocks.RenderWith(env, `<!-- wp:site-title {"isLink":false,"level":0} /-->`)
	if strings.Contains(out, "<a ") {
		t.Fatalf("expected no link, got %q", out)
	}
	if !strings.Contains(out, `<p class="wp-block-site-title">GoPress Fixture</p>`) {
		t.Fatalf("expected paragraph level 0, got %q", out)
	}
}

func TestRenderSiteTagline(t *testing.T) {
	env := &blocks.Env{
		SiteTagline: func() string { return "Just another site" },
	}
	out := blocks.RenderWith(env, `<!-- wp:site-tagline {"textAlign":"center"} /-->`)
	want := `<p class="wp-block-site-tagline has-text-align-center">Just another site</p>`
	if !strings.Contains(out, want) {
		t.Fatalf("got %q want substring %q", out, want)
	}
	empty := blocks.RenderWith(&blocks.Env{SiteTagline: func() string { return "" }}, `<!-- wp:site-tagline /-->`)
	if strings.TrimSpace(empty) != "" {
		t.Fatalf("empty tagline should omit markup, got %q", empty)
	}
}

func TestRenderSiteLogo(t *testing.T) {
	env := &blocks.Env{
		SiteLogoURL: func() string { return "http://127.0.0.1:8081/wp-content/uploads/logo.png" },
		HomeURL:     func() string { return "http://127.0.0.1:8081" },
	}
	out := blocks.RenderWith(env, `<!-- wp:site-logo {"width":80} /-->`)
	if !strings.Contains(out, `class="wp-block-site-logo"`) {
		t.Fatalf("missing wrapper: %q", out)
	}
	if !strings.Contains(out, `class="custom-logo-link"`) || !strings.Contains(out, `rel="home"`) {
		t.Fatalf("expected logo link, got %q", out)
	}
	if !strings.Contains(out, `width="80"`) || !strings.Contains(out, `src="http://127.0.0.1:8081/wp-content/uploads/logo.png"`) {
		t.Fatalf("expected sized logo img, got %q", out)
	}
	noLogo := blocks.RenderWith(&blocks.Env{}, `<!-- wp:site-logo /-->`)
	if strings.TrimSpace(noLogo) != "" {
		t.Fatalf("missing logo URL should omit markup, got %q", noLogo)
	}
}

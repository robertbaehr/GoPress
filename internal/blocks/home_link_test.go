package blocks_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/blocks"
)

func TestRenderHomeLinkDefaults(t *testing.T) {
	env := &blocks.Env{
		HomeURL:     func() string { return "http://example.test" },
		IsFrontPage: func() bool { return false },
	}
	out := blocks.RenderWith(env, `<!-- wp:home-link /-->`)
	if !strings.Contains(out, `class="wp-block-home-link wp-block-navigation-item"`) {
		t.Fatalf("missing li classes: %q", out)
	}
	if !strings.Contains(out, `class="wp-block-home-link__content wp-block-navigation-item__content"`) {
		t.Fatalf("missing link classes: %q", out)
	}
	if !strings.Contains(out, `href="http://example.test" rel="home"`) {
		t.Fatalf("expected home href+rel: %q", out)
	}
	if !strings.Contains(out, `>Home</a>`) {
		t.Fatalf("expected default label Home: %q", out)
	}
	if strings.Contains(out, `current-menu-item`) || strings.Contains(out, `aria-current`) {
		t.Fatalf("non-front should not be current: %q", out)
	}
}

func TestRenderHomeLinkCurrentFront(t *testing.T) {
	env := &blocks.Env{
		HomeURL:     func() string { return "http://example.test/" },
		IsFrontPage: func() bool { return true },
	}
	out := blocks.RenderWith(env,
		`<!-- wp:home-link {"label":"Start","opensInNewTab":true,"description":"Site home"} /-->`)
	if !strings.Contains(out, `current-menu-item`) {
		t.Fatalf("expected current-menu-item: %q", out)
	}
	if !strings.Contains(out, `aria-current="page"`) {
		t.Fatalf("expected aria-current: %q", out)
	}
	if !strings.Contains(out, `target="_blank"`) {
		t.Fatalf("expected target=_blank: %q", out)
	}
	if !strings.Contains(out, `>Start<span class="wp-block-navigation-item__description">Site home</span></a>`) {
		t.Fatalf("expected label+description: %q", out)
	}
	if !strings.Contains(out, `href="http://example.test"`) {
		t.Fatalf("expected trimmed home URL: %q", out)
	}
}

func TestRenderHomeLinkInsideNavigation(t *testing.T) {
	env := &blocks.Env{
		HomeURL:     func() string { return "http://example.test" },
		IsFrontPage: func() bool { return false },
	}
	out := blocks.RenderWith(env,
		`<!-- wp:navigation --><!-- wp:home-link {"label":"Home"} /--><!-- /wp:navigation -->`)
	if !strings.Contains(out, `wp-block-navigation`) {
		t.Fatalf("expected nav wrapper: %q", out)
	}
	if !strings.Contains(out, `wp-block-home-link`) {
		t.Fatalf("expected home-link inside nav: %q", out)
	}
}

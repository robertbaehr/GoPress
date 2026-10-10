package blocks_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/blocks"
)

func TestRenderNavigationLink(t *testing.T) {
	in := `<!-- wp:navigation-link {"label":"Home","type":"custom","url":"/"} /-->`
	out := blocks.Render(in)
	if !strings.Contains(out, `href="/"`) {
		t.Fatalf("missing href: %q", out)
	}
	if !strings.Contains(out, `Home`) {
		t.Fatalf("missing label: %q", out)
	}
	if !strings.Contains(out, `wp-block-navigation-item`) {
		t.Fatalf("missing class: %q", out)
	}
}

func TestRenderNavigationFromRef(t *testing.T) {
	env := &blocks.Env{
		LoadNavigation: func(ref int64) (string, bool) {
			if ref != 446 {
				return "", false
			}
			return `<!-- wp:navigation-link {"label":"Home","type":"custom","url":"/"} /-->`, true
		},
	}
	in := `<!-- wp:navigation {"ref":446} /-->`
	out := blocks.RenderWith(env, in)
	if !strings.Contains(out, `wp-block-navigation`) || !strings.Contains(out, `<nav`) {
		t.Fatalf("missing nav: %q", out)
	}
	if !strings.Contains(out, `>Home</span>`) {
		t.Fatalf("missing Home link: %q", out)
	}
}

func TestRenderNavigationPageListFallback(t *testing.T) {
	env := &blocks.Env{
		ListPages: func() []blocks.NavLink {
			return []blocks.NavLink{{Label: "Sample Page", URL: "/sample-page/"}}
		},
	}
	in := `<!-- wp:navigation -->
<!-- wp:page-list /-->
<!-- /wp:navigation -->`
	out := blocks.RenderWith(env, in)
	if !strings.Contains(out, "Sample Page") {
		t.Fatalf("missing page: %q", out)
	}
	if !strings.Contains(out, `href="/sample-page/"`) {
		t.Fatalf("missing url: %q", out)
	}
}

func TestRenderNavigationEmptyPagesFallback(t *testing.T) {
	env := &blocks.Env{
		ListPages: func() []blocks.NavLink {
			return []blocks.NavLink{{Label: "About", URL: "/about/"}}
		},
	}
	out := blocks.RenderWith(env, `<!-- wp:navigation /-->`)
	if !strings.Contains(out, "About") {
		t.Fatalf("expected pages fallback: %q", out)
	}
}

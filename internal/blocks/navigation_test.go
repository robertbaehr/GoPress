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

func TestRenderNavigationLinkPolish(t *testing.T) {
	env := &blocks.Env{CurrentPageID: 42}
	out := blocks.RenderWith(env,
		`<!-- wp:navigation-link {"label":"About","url":"/about/","id":42,"opensInNewTab":true,"rel":"nofollow","title":"About us","description":"Company"} /-->`)
	for _, want := range []string{
		`current-menu-item`,
		`aria-current="page"`,
		`target="_blank"`,
		`rel="nofollow"`,
		`title="About us"`,
		`wp-block-navigation-item__description`,
		`Company`,
		`href="/about/"`,
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in %q", want, out)
		}
	}
}

func TestRenderNavigationSubmenu(t *testing.T) {
	// Standalone submenu (no Navigation parent): showSubmenuIcon defaults true → toggle button.
	in := `<!-- wp:navigation-submenu {"label":"Parent","url":"/parent/"} -->
<!-- wp:navigation-link {"label":"Child","url":"/child/"} /-->
<!-- /wp:navigation-submenu -->`
	out := blocks.Render(in)
	for _, want := range []string{
		`has-child`,
		`wp-block-navigation__submenu-container`,
		`wp-block-navigation__submenu-icon`,
		`wp-block-navigation-submenu__toggle`,
		`>Parent</span>`,
		`href="/child/"`,
		`Child`,
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in %q", want, out)
		}
	}
}

func TestRenderNavigationLayoutAndSubmenuContext(t *testing.T) {
	// P5.117: justifyContent / flexWrap / textDecoration / className / open-on-click.
	in := `<!-- wp:navigation {"className":"main-nav","layout":{"orientation":"horizontal","justifyContent":"center","flexWrap":"nowrap"},"style":{"typography":{"textDecoration":"underline"}},"openSubmenusOnClick":true,"showSubmenuIcon":true,"anchor":"site-nav"} -->
<!-- wp:navigation-submenu {"label":"Parent","url":"/parent/"} -->
<!-- wp:navigation-link {"label":"Child","url":"/child/"} /-->
<!-- /wp:navigation-submenu -->
<!-- /wp:navigation -->`
	out := blocks.Render(in)
	for _, want := range []string{
		`items-justified-center`,
		`no-wrap`,
		`has-text-decoration-underline`,
		`main-nav`,
		`id="site-nav"`,
		`open-on-click`,
		`wp-block-navigation-submenu__toggle`,
		`aria-expanded="false"`,
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in %q", want, out)
		}
	}
	// showSubmenuIcon false omits hover toggle button (keeps open-on-click span icon).
	out2 := blocks.Render(`<!-- wp:navigation {"showSubmenuIcon":false} -->
<!-- wp:navigation-submenu {"label":"Parent","url":"/parent/"} -->
<!-- wp:navigation-link {"label":"Child","url":"/child/"} /-->
<!-- /wp:navigation-submenu -->
<!-- /wp:navigation -->`)
	if strings.Contains(out2, `wp-block-navigation-submenu__toggle`) {
		t.Fatalf("expected no submenu toggle when showSubmenuIcon=false: %q", out2)
	}
	if !strings.Contains(out2, `href="/parent/"`) {
		t.Fatalf("expected anchor when not open-on-click: %q", out2)
	}
}

func TestRenderNavigationLinkNofollowAttr(t *testing.T) {
	out := blocks.Render(`<!-- wp:navigation-link {"label":"Ext","url":"https://example.com","nofollow":true} /-->`)
	if !strings.Contains(out, `rel="nofollow"`) {
		t.Fatalf("expected nofollow: %q", out)
	}
}

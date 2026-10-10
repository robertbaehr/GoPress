package blocks_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/blocks"
)

func TestRenderPageListStandaloneClasses(t *testing.T) {
	env := &blocks.Env{
		ListPages: func() []blocks.NavLink {
			return []blocks.NavLink{
				{ID: 2, Label: "Sample Page", URL: "/sample-page/", Parent: 0},
				{ID: 10, Label: "About", URL: "/about/", Parent: 0},
			}
		},
	}
	out := blocks.RenderWith(env, `<!-- wp:page-list /-->`)
	if !strings.Contains(out, `class="wp-block-page-list"`) {
		t.Fatalf("missing wrapper: %q", out)
	}
	if !strings.Contains(out, `wp-block-pages-list__item`) {
		t.Fatalf("missing item class: %q", out)
	}
	if !strings.Contains(out, `wp-block-pages-list__item__link`) {
		t.Fatalf("missing link class: %q", out)
	}
	if !strings.Contains(out, `href="/sample-page/"`) || !strings.Contains(out, "About") {
		t.Fatalf("missing pages: %q", out)
	}
}

func TestRenderPageListNestedAndParentFilter(t *testing.T) {
	env := &blocks.Env{
		ListPages: func() []blocks.NavLink {
			return []blocks.NavLink{
				{ID: 436, Label: "Fixture Parent", URL: "/fixture-parent/", Parent: 0},
				{ID: 437, Label: "Fixture Child", URL: "/fixture-child/", Parent: 436},
				{ID: 438, Label: "Fixture Grandchild", URL: "/fixture-grandchild/", Parent: 437},
				{ID: 2, Label: "Sample Page", URL: "/sample-page/", Parent: 0},
			}
		},
		CurrentPageID: 437,
		FrontPageID:   440,
	}
	out := blocks.RenderWith(env, `<!-- wp:page-list /-->`)
	if !strings.Contains(out, `wp-block-page-list`) {
		t.Fatalf("expected nested ul class: %q", out)
	}
	if !strings.Contains(out, "Fixture Child") || !strings.Contains(out, "Fixture Grandchild") {
		t.Fatalf("expected nested children: %q", out)
	}
	if !strings.Contains(out, `current-menu-item`) || !strings.Contains(out, `aria-current="page"`) {
		t.Fatalf("expected current on child: %q", out)
	}
	if !strings.Contains(out, `has-child`) {
		t.Fatalf("expected has-child on parent: %q", out)
	}

	// parentPageID=436 → only children of 436 (and their descendants).
	out = blocks.RenderWith(env, `<!-- wp:page-list {"parentPageID":436} /-->`)
	if strings.Contains(out, "Sample Page") || strings.Contains(out, "Fixture Parent") {
		t.Fatalf("parent filter should omit siblings/self: %q", out)
	}
	if !strings.Contains(out, "Fixture Child") || !strings.Contains(out, "Fixture Grandchild") {
		t.Fatalf("parent filter should list child tree: %q", out)
	}

	out = blocks.RenderWith(env, `<!-- wp:page-list {"parentPageID":999} /-->`)
	if strings.TrimSpace(out) != "" {
		t.Fatalf("empty parent should render nothing: %q", out)
	}
}

func TestRenderPageListInsideNavigation(t *testing.T) {
	env := &blocks.Env{
		ListPages: func() []blocks.NavLink {
			return []blocks.NavLink{
				{ID: 440, Label: "Home", URL: "/", Parent: 0},
				{ID: 2, Label: "Sample Page", URL: "/sample-page/", Parent: 0},
			}
		},
		FrontPageID: 440,
	}
	out := blocks.RenderWith(env,
		`<!-- wp:navigation --><!-- wp:page-list /--><!-- /wp:navigation -->`)
	if !strings.Contains(out, `wp-block-navigation-item`) {
		t.Fatalf("expected nav item class: %q", out)
	}
	if !strings.Contains(out, `wp-block-navigation-item__content`) {
		t.Fatalf("expected nav link class: %q", out)
	}
	if !strings.Contains(out, `menu-item-home`) {
		t.Fatalf("expected menu-item-home on front page: %q", out)
	}
}

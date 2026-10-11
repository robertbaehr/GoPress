package blocks_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/blocks"
)

func TestRenderWidgetGroupP5132TitleAndSidebarStillEmpty(t *testing.T) {
	cases := []string{
		`<!-- wp:widget-group /-->`,
		`<!-- wp:widget-group {"title":""} /-->`,
		`<!-- wp:widget-group {"title":"Sidebar"} /-->`,
		`<!-- wp:widget-group {"title":"Main"} --><div class="wp-widget-group__inner-blocks"><!-- wp:paragraph --><p>Hi</p><!-- /wp:paragraph --></div><!-- /wp:widget-group -->`,
	}
	for _, raw := range cases {
		out := blocks.RenderWith(&blocks.Env{SidebarBeingRendered: "sidebar-1"}, raw)
		if strings.TrimSpace(out) != "" {
			t.Fatalf("widget-group should stay empty with sidebar note, got %q for %s", out, raw)
		}
		out = blocks.Render(raw)
		if strings.TrimSpace(out) != "" {
			t.Fatalf("widget-group should stay empty without sidebar, got %q for %s", out, raw)
		}
	}
}

func TestRenderWidgetGroupP5132DoesNotLeakTitleOrInners(t *testing.T) {
	raw := `<!-- wp:widget-group {"title":"Archives"} -->` +
		`<div class="wp-widget-group__inner-blocks">` +
		`<!-- wp:paragraph --><p class="leak">should not render</p><!-- /wp:paragraph -->` +
		`<!-- wp:legacy-widget {"idBase":"archives"} /-->` +
		`</div><!-- /wp:widget-group -->`
	env := &blocks.Env{SidebarBeingRendered: "footer-1"}
	out := blocks.RenderWith(env, raw)
	for _, bad := range []string{"wp-widget-group", "Archives", "should not render", "widget-title", "leak"} {
		if strings.Contains(out, bad) {
			t.Fatalf("expected empty placeholder, found %q in %q", bad, out)
		}
	}
}

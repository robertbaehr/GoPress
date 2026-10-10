package blocks_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/blocks"
)

func TestRenderWidgetGroupEmpty(t *testing.T) {
	cases := []string{
		`<!-- wp:widget-group /-->`,
		`<!-- wp:widget-group {"title":"Sidebar"} /-->`,
		`<!-- wp:widget-group {"title":"Footer"} --><div class="wp-widget-group__inner-blocks"></div><!-- /wp:widget-group -->`,
		`<!-- wp:widget-group {"title":"Main"} --><div class="wp-widget-group__inner-blocks"><!-- wp:paragraph --><p>Hi</p><!-- /wp:paragraph --></div><!-- /wp:widget-group -->`,
	}
	for _, raw := range cases {
		out := blocks.Render(raw)
		if strings.TrimSpace(out) != "" {
			t.Fatalf("widget-group should render empty, got %q for %s", out, raw)
		}
	}
}

func TestRenderWidgetGroupDoesNotLeakInnerHTML(t *testing.T) {
	raw := `<!-- wp:widget-group {"title":"Archives"} -->` +
		`<div class="wp-widget-group__inner-blocks">` +
		`<!-- wp:legacy-widget {"idBase":"archives"} /-->` +
		`<p class="widget">should not render</p>` +
		`</div><!-- /wp:widget-group -->`
	out := blocks.Render(raw)
	if strings.Contains(out, "wp-widget-group") || strings.Contains(out, "Archives") ||
		strings.Contains(out, "should not render") {
		t.Fatalf("expected empty placeholder, got %q", out)
	}
}

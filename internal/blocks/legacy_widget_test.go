package blocks_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/blocks"
)

func TestRenderLegacyWidgetEmpty(t *testing.T) {
	cases := []string{
		`<!-- wp:legacy-widget /-->`,
		`<!-- wp:legacy-widget {"idBase":"search"} /-->`,
		`<!-- wp:legacy-widget {"id":"search-2","idBase":"search"} /-->`,
		`<!-- wp:legacy-widget {"idBase":"text","instance":{"encoded":"YToxOntzOjQ6InRleHQiO3M6NDoiaGVsbCI7fQ==","hash":"deadbeef"}} /-->`,
		`<!-- wp:legacy-widget {"idBase":"calendar"} --><div class="widget">should not render</div><!-- /wp:legacy-widget -->`,
	}
	for _, raw := range cases {
		out := blocks.Render(raw)
		if strings.TrimSpace(out) != "" {
			t.Fatalf("legacy-widget should render empty, got %q for %s", out, raw)
		}
	}
}

func TestRenderLegacyWidgetDoesNotLeakInnerHTML(t *testing.T) {
	raw := `<!-- wp:legacy-widget {"idBase":"archives"} -->` +
		`<aside class="widget widget_archive"><ul><li>Jan</li></ul></aside>` +
		`<!-- /wp:legacy-widget -->`
	out := blocks.Render(raw)
	if strings.Contains(out, "widget_archive") || strings.Contains(out, "Jan") {
		t.Fatalf("expected empty placeholder, got %q", out)
	}
}

package blocks_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/blocks"
)

func TestCompileStyleColorSpacingTypography(t *testing.T) {
	style := map[string]any{
		"color": map[string]any{
			"background": "#abcdef",
			"text":       "#111111",
		},
		"spacing": map[string]any{
			"padding": map[string]any{
				"top":    "var:preset|spacing|40",
				"bottom": "20px",
			},
			"blockGap": "1rem",
		},
		"typography": map[string]any{
			"fontSize":   "1.25rem",
			"lineHeight": "1.6",
		},
		"border": map[string]any{
			"width":  "1px",
			"style":  "solid",
			"color":  "#ccc",
			"radius": "4px",
		},
	}
	got := blocks.CompileStyle(style)
	for _, want := range []string{
		"background-color:#abcdef",
		"color:#111111",
		"padding-top:var(--wp--preset--spacing--40)",
		"padding-bottom:20px",
		"gap:1rem",
		"font-size:1.25rem",
		"line-height:1.6",
		"border-width:1px",
		"border-style:solid",
		"border-color:#ccc",
		"border-radius:4px",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %q in %q", want, got)
		}
	}
}

func TestCompileCSSRule(t *testing.T) {
	rule := blocks.CompileCSS(".wp-block-group", map[string]any{
		"color": map[string]any{"background": "#fff"},
	})
	want := ".wp-block-group{background-color:#fff}"
	if rule != want {
		t.Fatalf("got %q want %q", rule, want)
	}
}

func TestSupportsUsesStyleEngine(t *testing.T) {
	in := `<!-- wp:group {"style":{"typography":{"fontSize":"18px"},"border":{"radius":"8px"}}} -->
<div class="wp-block-group">X</div>
<!-- /wp:group -->`
	out := blocks.Render(in)
	if !strings.Contains(out, "font-size:18px") {
		t.Fatalf("missing font-size from style engine: %q", out)
	}
	if !strings.Contains(out, "border-radius:8px") {
		t.Fatalf("missing border-radius from style engine: %q", out)
	}
}

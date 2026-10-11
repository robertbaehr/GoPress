package blocks_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/blocks"
)

// P5.167: core/categories coerces align left|center|right|wide|full; drops none/bogus.
func TestRenderCategoriesAttrsAlignP5167(t *testing.T) {
	env := &blocks.Env{
		ListCategories: func(_ string) []blocks.CategoryItem {
			return []blocks.CategoryItem{{Name: "News", URL: "http://example.test/cat/news/", Count: 1, ID: 1}}
		},
	}
	for _, align := range []string{"left", "center", "right", "wide", "full"} {
		out := blocks.RenderWith(env, `<!-- wp:categories {"align":"`+align+`"} /-->`)
		want := "align" + align
		if !strings.Contains(out, want) {
			t.Fatalf("expected %q in %q", want, out)
		}
	}
	for _, bad := range []string{"none", "bogus"} {
		out := blocks.RenderWith(env, `<!-- wp:categories {"align":"`+bad+`"} /-->`)
		if strings.Contains(out, "align"+bad) {
			t.Fatalf("align=%s should be dropped: %q", bad, out)
		}
	}
}

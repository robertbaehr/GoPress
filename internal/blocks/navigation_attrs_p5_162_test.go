package blocks_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/blocks"
)

// P5.162: core/navigation coerces align to wide|full; drops left/center/right/none/bogus.
func TestRenderNavigationAttrsAlignP5162(t *testing.T) {
	inner := `<!-- wp:navigation-link {"label":"Home","url":"/","kind":"custom"} /-->`
	for _, align := range []string{"wide", "full"} {
		out := blocks.Render(`<!-- wp:navigation {"align":"` + align + `"} -->` + inner + `<!-- /wp:navigation -->`)
		want := "align" + align
		if !strings.Contains(out, want) {
			t.Fatalf("expected %q in %q", want, out)
		}
	}
	for _, bad := range []string{"left", "center", "right", "none", "bogus"} {
		out := blocks.Render(`<!-- wp:navigation {"align":"` + bad + `"} -->` + inner + `<!-- /wp:navigation -->`)
		if strings.Contains(out, "align"+bad) {
			t.Fatalf("align=%s should be dropped: %q", bad, out)
		}
	}
}

package blocks_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/blocks"
)

// P5.163: core/search coerces align to left|center|right; drops wide/full/none/bogus.
func TestRenderSearchAttrsAlignP5163(t *testing.T) {
	for _, align := range []string{"left", "center", "right"} {
		out := blocks.Render(`<!-- wp:search {"align":"` + align + `","label":"Search"} /-->`)
		want := "align" + align
		if !strings.Contains(out, want) {
			t.Fatalf("expected %q in %q", want, out)
		}
	}
	for _, bad := range []string{"wide", "full", "none", "bogus"} {
		out := blocks.Render(`<!-- wp:search {"align":"` + bad + `","label":"Search"} /-->`)
		if strings.Contains(out, "align"+bad) {
			t.Fatalf("align=%s should be dropped: %q", bad, out)
		}
	}
}

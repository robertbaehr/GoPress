package blocks_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/blocks"
)

// P5.166: core/archives coerces align left|center|right|wide|full; drops none/bogus.
func TestRenderArchivesAttrsAlignP5166(t *testing.T) {
	env := &blocks.Env{
		ListArchives: func(_ string) []blocks.ArchiveItem {
			return []blocks.ArchiveItem{{URL: "http://example.test/2024/01/", Label: "January 2024"}}
		},
	}
	for _, align := range []string{"left", "center", "right", "wide", "full"} {
		out := blocks.RenderWith(env, `<!-- wp:archives {"align":"`+align+`"} /-->`)
		want := "align" + align
		if !strings.Contains(out, want) {
			t.Fatalf("expected %q in %q", want, out)
		}
	}
	for _, bad := range []string{"none", "bogus"} {
		out := blocks.RenderWith(env, `<!-- wp:archives {"align":"`+bad+`"} /-->`)
		if strings.Contains(out, "align"+bad) {
			t.Fatalf("align=%s should be dropped: %q", bad, out)
		}
	}
}

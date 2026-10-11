package blocks_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/blocks"
)

// P5.168: core/latest-posts coerces align left|center|right|wide|full; drops none/bogus.
func TestRenderLatestPostsAttrsAlignP5168(t *testing.T) {
	env := &blocks.Env{
		ListPosts: func(_ blocks.QueryLoopArgs) []blocks.QueryPost {
			return []blocks.QueryPost{{ID: 1, Title: "Hello", Link: "http://example.test/?p=1", Date: "2024-01-01"}}
		},
	}
	for _, align := range []string{"left", "center", "right", "wide", "full"} {
		out := blocks.RenderWith(env, `<!-- wp:latest-posts {"align":"`+align+`"} /-->`)
		want := "align" + align
		if !strings.Contains(out, want) {
			t.Fatalf("expected %q in %q", want, out)
		}
	}
	for _, bad := range []string{"none", "bogus"} {
		out := blocks.RenderWith(env, `<!-- wp:latest-posts {"align":"`+bad+`"} /-->`)
		if strings.Contains(out, "align"+bad) {
			t.Fatalf("align=%s should be dropped: %q", bad, out)
		}
	}
}

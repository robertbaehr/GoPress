package blocks_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/blocks"
)

// P5.165: core/calendar coerces align left|center|right|wide|full; drops none/bogus.
func TestRenderCalendarAttrsAlignP5165(t *testing.T) {
	env := &blocks.Env{
		HasPublishedPosts: func() bool { return true },
		HomeURL:           func() string { return "http://example.test" },
	}
	for _, align := range []string{"left", "center", "right", "wide", "full"} {
		out := blocks.RenderWith(env, `<!-- wp:calendar {"align":"`+align+`"} /-->`)
		want := "align" + align
		if !strings.Contains(out, want) {
			t.Fatalf("expected %q in %q", want, out)
		}
	}
	for _, bad := range []string{"none", "bogus"} {
		out := blocks.RenderWith(env, `<!-- wp:calendar {"align":"`+bad+`"} /-->`)
		if strings.Contains(out, "align"+bad) {
			t.Fatalf("align=%s should be dropped: %q", bad, out)
		}
	}
}

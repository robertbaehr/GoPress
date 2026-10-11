package blocks_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/blocks"
)

// P5.164: core/loginout coerces align to left|center|right; drops wide/full/none/bogus.
func TestRenderLoginoutAttrsAlignP5164(t *testing.T) {
	env := &blocks.Env{
		IsLoggedIn: func() bool { return false },
		LoginURL:   func(string) string { return "/wp-login.php" },
	}
	for _, align := range []string{"left", "center", "right"} {
		out := blocks.RenderWith(env, `<!-- wp:loginout {"align":"`+align+`"} /-->`)
		want := "align" + align
		if !strings.Contains(out, want) {
			t.Fatalf("expected %q in %q", want, out)
		}
	}
	for _, bad := range []string{"wide", "full", "none", "bogus"} {
		out := blocks.RenderWith(env, `<!-- wp:loginout {"align":"`+bad+`"} /-->`)
		if strings.Contains(out, "align"+bad) {
			t.Fatalf("align=%s should be dropped: %q", bad, out)
		}
	}
}

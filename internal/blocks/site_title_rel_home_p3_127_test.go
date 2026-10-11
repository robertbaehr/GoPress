package blocks_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/blocks"
)

// P3.127: core/site-title link attrs match WP href/target/rel then aria-current.
func TestRenderSiteTitleRelHomeAttrOrder(t *testing.T) {
	env := &blocks.Env{
		SiteName:    func() string { return "GoPress Fixture" },
		HomeURL:     func() string { return "http://127.0.0.1:8081" },
		IsFrontPage: func() bool { return true },
		IsPaged:     func() bool { return false },
	}
	out := blocks.RenderWith(env, `<!-- wp:site-title /-->`)
	want := `<a href="http://127.0.0.1:8081" target="_self" rel="home" aria-current="page">GoPress Fixture</a>`
	if !strings.Contains(out, want) {
		t.Fatalf("attr order mismatch: got %q want substring %q", out, want)
	}
	// aria-current must not precede href (pre-P3.127 bug).
	if i := strings.Index(out, `aria-current="page"`); i >= 0 {
		if j := strings.Index(out, `href="`); j < 0 || i < j {
			t.Fatalf("aria-current must follow href: %q", out)
		}
	}
}

package blocks_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/blocks"
)

// P5.135: home-link inherits Navigation context colors, emits anchor id,
// and keeps href/rel/target/aria-current attribute order.
func TestRenderHomeLinkAttrsFromNavContext(t *testing.T) {
	env := &blocks.Env{
		HomeURL:     func() string { return "http://example.test" },
		IsFrontPage: func() bool { return true },
	}
	out := blocks.RenderWith(env, `<!-- wp:navigation {"textColor":"vivid-red","backgroundColor":"cyan-bluish-gray"} -->
<!-- wp:home-link {"label":"Home","anchor":"nav-home","opensInNewTab":true} /-->
<!-- /wp:navigation -->`)
	if !strings.Contains(out, `id="nav-home"`) {
		t.Fatalf("expected anchor id: %q", out)
	}
	if !strings.Contains(out, `has-vivid-red-color`) || !strings.Contains(out, `has-cyan-bluish-gray-background-color`) {
		t.Fatalf("expected nav context colors on home-link: %q", out)
	}
	// Class order: wp-block-home-link … colors … wp-block-navigation-item current-menu-item
	li := out[strings.Index(out, `<li class="`):]
	li = li[:strings.Index(li, `>`)+1]
	homeIdx := strings.Index(li, "wp-block-home-link")
	colorIdx := strings.Index(li, "has-vivid-red-color")
	navIdx := strings.Index(li, "wp-block-navigation-item")
	curIdx := strings.Index(li, "current-menu-item")
	if homeIdx < 0 || colorIdx < 0 || navIdx < 0 || curIdx < 0 {
		t.Fatalf("missing class tokens in %q", li)
	}
	if !(homeIdx < colorIdx && colorIdx < navIdx && navIdx < curIdx) {
		t.Fatalf("class order mismatch: %q", li)
	}
	want := `href="http://example.test" rel="home" target="_blank" aria-current="page"`
	if !strings.Contains(out, want) {
		t.Fatalf("attr order mismatch: got %q want substring %q", out, want)
	}
}

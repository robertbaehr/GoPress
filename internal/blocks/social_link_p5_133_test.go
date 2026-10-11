package blocks_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/blocks"
)

func TestRenderSocialLinkExtraServicesP5133(t *testing.T) {
	// P5.133: deepen catalog — names + brand classes for previously generic services.
	cases := []struct {
		service string
		label   string
	}{
		{"spotify", "Spotify"},
		{"tumblr", "Tumblr"},
		{"vimeo", "Vimeo"},
		{"chain", "Link"},
		{"feed", "RSS Feed"},
		{"fivehundredpx", "500px"},
		{"soundcloud", "SoundCloud"},
	}
	for _, tc := range cases {
		src := `<!-- wp:social-link {"url":"https://example.com/` + tc.service + `","service":"` + tc.service + `"} /-->`
		out := blocks.Render(src)
		for _, needle := range []string{
			`wp-social-link-` + tc.service,
			tc.label,
			`<svg`,
			`href="https://example.com/` + tc.service + `"`,
		} {
			if !strings.Contains(out, needle) {
				t.Fatalf("%s: missing %q in %q", tc.service, needle, out)
			}
		}
	}
}

func TestRenderSocialLinkShareNameP5133(t *testing.T) {
	out := blocks.Render(`<!-- wp:social-link {"url":"https://example.com/share","service":"share"} /-->`)
	if !strings.Contains(out, "Share Icon") {
		t.Fatalf("expected Share Icon label: %q", out)
	}
}

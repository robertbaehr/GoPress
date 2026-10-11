package blocks_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/blocks"
)

// P5.160: core/embed coerces align; drops none/bogus; keeps allowed.
func TestRenderEmbedAttrsAlignP5160(t *testing.T) {
	for _, align := range []string{"left", "center", "right", "wide", "full"} {
		out := blocks.Render(`<!-- wp:embed {"url":"https://www.youtube.com/watch?v=dQw4w9WgXcQ","align":"` + align + `"} /-->`)
		want := "align" + align
		if !strings.Contains(out, want) {
			t.Fatalf("expected %q in %q", want, out)
		}
	}
	none := blocks.Render(`<!-- wp:embed {"url":"https://www.youtube.com/watch?v=dQw4w9WgXcQ","align":"none"} /-->`)
	if strings.Contains(none, "alignnone") {
		t.Fatalf("align=none should be dropped: %q", none)
	}
	bogus := blocks.Render(`<!-- wp:embed {"url":"https://www.youtube.com/watch?v=dQw4w9WgXcQ","align":"bogus"} /-->`)
	if strings.Contains(bogus, "alignbogus") {
		t.Fatalf("align=bogus should be dropped: %q", bogus)
	}
}

func TestRenderEmbedAttrsAnchorClassP5160(t *testing.T) {
	out := blocks.Render(`<!-- wp:embed {"url":"https://youtu.be/dQw4w9WgXcQ","className":"is-style-default","anchor":"yt-embed","align":"wide"} /-->`)
	for _, want := range []string{
		`wp-block-embed`,
		`alignwide`,
		`is-style-default`,
		`id="yt-embed"`,
		`is-provider-youtube`,
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in %q", want, out)
		}
	}
}

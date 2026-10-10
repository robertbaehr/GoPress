package blocks_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/blocks"
)

func TestRenderHomeLinkColorsAndClassName(t *testing.T) {
	env := &blocks.Env{
		HomeURL:     func() string { return "http://example.test" },
		IsFrontPage: func() bool { return false },
	}
	out := blocks.RenderWith(env,
		`<!-- wp:home-link {"label":"Home","textColor":"vivid-red","backgroundColor":"cyan-bluish-gray","className":"is-style-pill","style":{"color":{"text":"#111111"}}} /-->`)
	for _, want := range []string{
		`wp-block-home-link`,
		`has-text-color`,
		`has-vivid-red-color`,
		`has-background`,
		`has-cyan-bluish-gray-background-color`,
		`is-style-pill`,
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in %q", want, out)
		}
	}
	// Named textColor wins over custom style color.
	if strings.Contains(out, `style="color:`) {
		t.Fatalf("named textColor should skip custom color style: %q", out)
	}
}

func TestRenderHomeLinkCustomColors(t *testing.T) {
	env := &blocks.Env{
		HomeURL:     func() string { return "http://example.test" },
		IsFrontPage: func() bool { return false },
	}
	out := blocks.RenderWith(env,
		`<!-- wp:home-link {"label":"Home","style":{"color":{"text":"#abcdef","background":"#fedcba"}}} /-->`)
	if !strings.Contains(out, `has-text-color`) || !strings.Contains(out, `has-background`) {
		t.Fatalf("expected color classes: %q", out)
	}
	if !strings.Contains(out, `style="color: #abcdef;background-color: #fedcba;"`) {
		t.Fatalf("expected custom color styles: %q", out)
	}
}

func TestRenderHomeLinkCurrentOnHomeEdge(t *testing.T) {
	env := &blocks.Env{
		HomeURL:         func() string { return "http://example.test" },
		IsFrontPage:     func() bool { return false },
		IsHome:          func() bool { return true },
		PageForPostsID:  func() int64 { return 7 },
		QueriedObjectID: func() int64 { return 0 },
	}
	out := blocks.RenderWith(env, `<!-- wp:home-link {"label":"Home"} /-->`)
	if !strings.Contains(out, `current-menu-item`) || !strings.Contains(out, `aria-current="page"`) {
		t.Fatalf("expected current on home edge: %q", out)
	}

	env.QueriedObjectID = func() int64 { return 7 }
	out = blocks.RenderWith(env, `<!-- wp:home-link {"label":"Home"} /-->`)
	if strings.Contains(out, `current-menu-item`) {
		t.Fatalf("posts page should not mark home-link current: %q", out)
	}
}

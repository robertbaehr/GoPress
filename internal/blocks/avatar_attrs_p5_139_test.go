package blocks_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/blocks"
)

// P5.139: core/avatar wrapper honors supports.align and supports.anchor.
func TestRenderAvatarAlignAnchorP5139(t *testing.T) {
	env := &blocks.Env{
		HomeURL:     func() string { return "http://example.test" },
		AuthorName:  func(int64) string { return "Ada" },
		AuthorEmail: func(int64) string { return "ada@example.com" },
		AuthorLink:  func(int64) string { return "http://example.test/author/ada/" },
	}
	env.SetCurrentPost(&blocks.QueryPost{ID: 1, Title: "Hello", Link: "/?p=1"})
	out := blocks.RenderWith(env, `<!-- wp:avatar {"size":48,"align":"left","anchor":"byline-avatar","className":"is-style-rounded"} /-->`)
	for _, want := range []string{
		`wp-block-avatar`,
		`alignleft`,
		`is-style-rounded`,
		`id="byline-avatar"`,
		`avatar-48`,
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in: %q", want, out)
		}
	}
}

func TestRenderAvatarEmptyWithoutEmailP5139(t *testing.T) {
	env := &blocks.Env{
		AuthorName:  func(int64) string { return "Ada" },
		AuthorEmail: func(int64) string { return "" },
	}
	env.SetCurrentPost(&blocks.QueryPost{ID: 1, Title: "Hello", Link: "/?p=1"})
	out := blocks.RenderWith(env, `<!-- wp:avatar {"anchor":"should-not-appear","align":"right"} /-->`)
	if strings.TrimSpace(out) != "" {
		t.Fatalf("expected empty without email, got %q", out)
	}
}

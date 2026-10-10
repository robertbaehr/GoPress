package blocks_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/blocks"
)

func TestRenderReadMore(t *testing.T) {
	env := &blocks.Env{}
	env.SetCurrentPost(&blocks.QueryPost{
		ID:    42,
		Title: "Hello World",
		Link:  "http://example.test/hello-world/",
	})
	out := blocks.RenderWith(env, `<!-- wp:read-more {"content":"Continue","justifyContent":"right","linkTarget":"_blank"} /-->`)
	checks := []string{
		`class="wp-block-read-more is-justified-right"`,
		`href="http://example.test/hello-world/"`,
		`target="_blank"`,
		`Continue`,
		`screen-reader-text`,
		`: Hello World`,
	}
	for _, want := range checks {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in %q", want, out)
		}
	}
}

func TestRenderReadMoreDefaultText(t *testing.T) {
	env := &blocks.Env{}
	env.SetCurrentPost(&blocks.QueryPost{ID: 1, Title: "", Link: "/?p=1"})
	out := blocks.RenderWith(env, `<!-- wp:read-more /-->`)
	if !strings.Contains(out, "Read more") {
		t.Fatalf("expected default link text: %q", out)
	}
	if !strings.Contains(out, "untitled post 1") {
		t.Fatalf("expected untitled fallback: %q", out)
	}
}

func TestRenderReadMoreEmptyWithoutPost(t *testing.T) {
	out := blocks.RenderWith(&blocks.Env{}, `<!-- wp:read-more /-->`)
	if strings.TrimSpace(out) != "" {
		t.Fatalf("expected empty without post: %q", out)
	}
}

func TestRenderMoreAndNextPage(t *testing.T) {
	more := blocks.RenderWith(&blocks.Env{}, `<!-- wp:more {"customText":"Keep reading","noTeaser":true} /-->`)
	if !strings.Contains(more, "<!--more Keep reading-->") {
		t.Fatalf("more marker: %q", more)
	}
	if !strings.Contains(more, "<!--noteaser-->") {
		t.Fatalf("noteaser: %q", more)
	}
	next := blocks.RenderWith(&blocks.Env{}, `<!-- wp:nextpage /-->`)
	if strings.TrimSpace(next) != "<!--nextpage-->" {
		t.Fatalf("nextpage: %q", next)
	}
}

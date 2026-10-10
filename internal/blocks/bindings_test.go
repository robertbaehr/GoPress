package blocks_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/blocks"
)

func TestPostMetaBindingReplacesParagraphContent(t *testing.T) {
	in := `<!-- wp:paragraph {"metadata":{"bindings":{"content":{"source":"core/post-meta","args":{"key":"subtitle"}}}}} -->
<p>Fallback</p>
<!-- /wp:paragraph -->`
	env := &blocks.Env{
		GetPostMeta: func(postID int64, key string) string {
			if postID == 7 && key == "subtitle" {
				return "Bound subtitle"
			}
			return ""
		},
	}
	env.SetCurrentPost(&blocks.QueryPost{ID: 7, Title: "T"})
	out := blocks.RenderWith(env, in)
	if !strings.Contains(out, "Bound subtitle") {
		t.Fatalf("expected bound meta content, got %q", out)
	}
	if strings.Contains(out, "Fallback") {
		t.Fatalf("fallback should be replaced: %q", out)
	}
}

func TestPatternOverridesBinding(t *testing.T) {
	in := `<!-- wp:paragraph {"metadata":{"name":"hero","bindings":{"__default":{"source":"core/pattern-overrides"}}}} -->
<p>Pattern fallback</p>
<!-- /wp:paragraph -->`
	env := &blocks.Env{
		PatternOverrides: map[string]map[string]string{
			"hero": {"content": "Override copy"},
		},
	}
	out := blocks.RenderWith(env, in)
	if !strings.Contains(out, "Override copy") {
		t.Fatalf("expected pattern override, got %q", out)
	}
}

func TestPostMetaBindingMissingKeyKeepsFallback(t *testing.T) {
	in := `<!-- wp:paragraph {"metadata":{"bindings":{"content":{"source":"core/post-meta","args":{"key":"missing"}}}}} -->
<p>Keep me</p>
<!-- /wp:paragraph -->`
	env := &blocks.Env{
		GetPostMeta: func(postID int64, key string) string { return "" },
	}
	env.SetCurrentPost(&blocks.QueryPost{ID: 1})
	out := blocks.RenderWith(env, in)
	if !strings.Contains(out, "Keep me") {
		t.Fatalf("expected fallback retained: %q", out)
	}
}

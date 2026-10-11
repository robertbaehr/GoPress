package blocks_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/blocks"
)

func TestPatternCircularHaltDebug(t *testing.T) {
	// Pattern A embeds itself via core/pattern slug.
	nested := `<!-- wp:pattern {"slug":"demo/loop"} /-->`
	env := &blocks.Env{
		DebugDisplay: true,
		LoadPattern: func(slug string) (string, bool) {
			if slug == "demo/loop" {
				return nested, true
			}
			return "", false
		},
	}
	out := blocks.RenderWith(env, `<!-- wp:pattern {"slug":"demo/loop"} /-->`)
	if !strings.Contains(out, `[block rendering halted for pattern "demo/loop"]`) {
		t.Fatalf("expected debug halt, got %q", out)
	}
}

func TestPatternCircularQuietWithoutDebug(t *testing.T) {
	nested := `<!-- wp:pattern {"slug":"demo/loop"} /-->`
	env := &blocks.Env{
		DebugDisplay: false,
		LoadPattern: func(slug string) (string, bool) {
			if slug == "demo/loop" {
				return nested, true
			}
			return "", false
		},
	}
	out := blocks.RenderWith(env, `<!-- wp:pattern {"slug":"demo/loop"} /-->`)
	if strings.Contains(out, "block rendering halted") {
		t.Fatalf("quiet mode should omit halt: %q", out)
	}
	if strings.TrimSpace(out) != "" {
		t.Fatalf("expected empty render without debug, got %q", out)
	}
}

func TestPatternMissingSlugEmpty(t *testing.T) {
	out := blocks.RenderWith(&blocks.Env{
		LoadPattern: func(slug string) (string, bool) { return "x", true },
	}, `<!-- wp:pattern /-->`)
	if strings.TrimSpace(out) != "" {
		t.Fatalf("missing slug should be empty: %q", out)
	}
}

func TestPatternMissingLoadEmpty(t *testing.T) {
	out := blocks.RenderWith(&blocks.Env{
		LoadPattern: func(slug string) (string, bool) { return "", false },
	}, `<!-- wp:pattern {"slug":"missing"} /-->`)
	if strings.TrimSpace(out) != "" {
		t.Fatalf("missing load should be empty: %q", out)
	}
}

func TestPatternRendersInnerParagraph(t *testing.T) {
	pattern := `<!-- wp:paragraph -->
<p>Pattern body</p>
<!-- /wp:paragraph -->`
	env := &blocks.Env{
		LoadPattern: func(slug string) (string, bool) {
			if slug == "demo/hello" {
				return pattern, true
			}
			return "", false
		},
	}
	out := blocks.RenderWith(env, `<!-- wp:pattern {"slug":"demo/hello"} /-->`)
	if !strings.Contains(out, "Pattern body") {
		t.Fatalf("expected pattern body, got %q", out)
	}
}

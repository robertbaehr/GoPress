package blocks_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/blocks"
)

func TestSyncedPatternCircularHaltDebug(t *testing.T) {
	// Pattern A embeds itself via core/block ref=7.
	pattern := `<!-- wp:block {"ref":7} /-->`
	in := `<!-- wp:block {"ref":7} /-->`
	env := &blocks.Env{
		DebugDisplay: true,
		LoadBlock: func(ref int64) (string, bool) {
			if ref == 7 {
				return pattern, true
			}
			return "", false
		},
	}
	out := blocks.RenderWith(env, in)
	if !strings.Contains(out, "[block rendering halted]") {
		t.Fatalf("expected debug halt, got %q", out)
	}
}

func TestSyncedPatternCircularQuietWithoutDebug(t *testing.T) {
	pattern := `<!-- wp:block {"ref":7} /-->`
	in := `<!-- wp:block {"ref":7} /-->`
	env := &blocks.Env{
		DebugDisplay: false,
		LoadBlock: func(ref int64) (string, bool) {
			if ref == 7 {
				return pattern, true
			}
			return "", false
		},
	}
	out := blocks.RenderWith(env, in)
	if strings.Contains(out, "[block rendering halted]") {
		t.Fatalf("quiet mode should omit halt: %q", out)
	}
	if strings.TrimSpace(out) != "" {
		t.Fatalf("expected empty render without debug, got %q", out)
	}
}

func TestSyncedPatternMissingRefEmpty(t *testing.T) {
	out := blocks.RenderWith(&blocks.Env{
		LoadBlock: func(ref int64) (string, bool) { return "x", true },
	}, `<!-- wp:block /-->`)
	if strings.TrimSpace(out) != "" {
		t.Fatalf("missing ref should be empty: %q", out)
	}
}

func TestSyncedPatternMissingLoadEmpty(t *testing.T) {
	out := blocks.RenderWith(&blocks.Env{
		LoadBlock: func(ref int64) (string, bool) { return "", false },
	}, `<!-- wp:block {"ref":99} /-->`)
	if strings.TrimSpace(out) != "" {
		t.Fatalf("missing load should be empty: %q", out)
	}
}

func TestSyncedPatternRendersInnerParagraph(t *testing.T) {
	pattern := `<!-- wp:paragraph -->
<p>Reusable body</p>
<!-- /wp:paragraph -->`
	env := &blocks.Env{
		LoadBlock: func(ref int64) (string, bool) {
			if ref == 3 {
				return pattern, true
			}
			return "", false
		},
	}
	out := blocks.RenderWith(env, `<!-- wp:block {"ref":3} /-->`)
	if !strings.Contains(out, "Reusable body") {
		t.Fatalf("expected pattern body, got %q", out)
	}
}

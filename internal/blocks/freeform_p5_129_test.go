package blocks_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/blocks"
)

func TestFreeformContentAttrWinsOverInnerHTML(t *testing.T) {
	raw := `<!-- wp:freeform {"content":"<p>From attr</p>"} -->
<p>From inner</p>
<!-- /wp:freeform -->`
	out := blocks.Render(raw)
	if out != "<p>From attr</p>" {
		t.Fatalf("content attr should win: %q", out)
	}
}

func TestFreeformEmptyContentAttrFallsThrough(t *testing.T) {
	raw := `<!-- wp:freeform {"content":""} -->
<p class="classic">Body</p>
<!-- /wp:freeform -->`
	out := blocks.Render(raw)
	if !strings.Contains(out, `class="classic"`) || !strings.Contains(out, "Body") {
		t.Fatalf("empty content attr should fall through: %q", out)
	}
}

func TestFreeformNullContentAttrFallsThrough(t *testing.T) {
	raw := `<!-- wp:freeform {"content":null} -->
<p>Null attr</p>
<!-- /wp:freeform -->`
	out := blocks.Render(raw)
	if !strings.Contains(out, "Null attr") {
		t.Fatalf("null content attr should fall through: %q", out)
	}
}

func TestNamelessClassicUsesFreeformPath(t *testing.T) {
	// Parser emits nameless freeform HTML for classic content without delimiters.
	out := blocks.Render(`<p class="wp-some-class">Testing freeform block</p>`)
	if !strings.Contains(out, "Testing freeform block") || !strings.Contains(out, "wp-some-class") {
		t.Fatalf("nameless classic: %q", out)
	}
}

func TestFreeformInnerContentJoin(t *testing.T) {
	name := "core/freeform"
	block := blocks.Block{
		BlockName:    &name,
		Attrs:        map[string]any{},
		InnerContent: []any{"<p>", "Joined", "</p>"},
	}
	out := blocks.RenderBlock(block)
	if out != "<p>Joined</p>" {
		t.Fatalf("innerContent join: %q", out)
	}
}

func TestFreeformSelfClosingContentAttr(t *testing.T) {
	raw := `<!-- wp:freeform {"content":"<div>Classic self-close</div>"} /-->`
	out := blocks.Render(raw)
	if out != "<div>Classic self-close</div>" {
		t.Fatalf("self-closing freeform: %q", out)
	}
}

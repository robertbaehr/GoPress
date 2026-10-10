package blocks_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/blocks"
)

func TestRenderHTMLFixture(t *testing.T) {
	out := blocks.Render(readSerialized(t, "core__html"))
	if !strings.Contains(out, "<h1>Some HTML code</h1>") || !strings.Contains(out, "<div>This is a div</div>") {
		t.Fatalf("html fixture: %q", out)
	}
	if strings.Contains(out, "<!-- wp:") {
		t.Fatalf("html leftover comments: %q", out)
	}
}

func TestRenderHTMLContentAttribute(t *testing.T) {
	raw := `<!-- wp:html {"content":"<p class=\"x\">From attr</p>"} /-->`
	out := blocks.Render(raw)
	if out != `<p class="x">From attr</p>` {
		t.Fatalf("html content attr: %q", out)
	}
}

func TestRenderFreeformFixture(t *testing.T) {
	out := blocks.Render(readSerialized(t, "core__freeform"))
	if !strings.Contains(out, "Testing freeform block") || !strings.Contains(out, "wp-some-class") {
		t.Fatalf("freeform fixture: %q", out)
	}
}

func TestRenderFreeformContentAttribute(t *testing.T) {
	raw := `<!-- wp:freeform {"content":"<p>Classic content</p>"} /-->`
	out := blocks.Render(raw)
	if out != "<p>Classic content</p>" {
		t.Fatalf("freeform content attr: %q", out)
	}
}

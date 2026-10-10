package blocks_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/blocks"
)

func TestRenderButtonSynthesizeAndWidth(t *testing.T) {
	src := `<!-- wp:button {"url":"https://example.com/go","text":"Click me","style":{"dimensions":{"width":"50%"}}} /-->`
	out := blocks.Render(src)
	if !strings.Contains(out, `class="wp-block-button`) {
		t.Fatalf("missing wrapper: %q", out)
	}
	if !strings.Contains(out, `href="https://example.com/go"`) || !strings.Contains(out, "Click me") {
		t.Fatalf("missing link/text: %q", out)
	}
	if !strings.Contains(out, "has-custom-width") || !strings.Contains(out, "wp-block-button__width-50") {
		t.Fatalf("missing width classes: %q", out)
	}
	if !strings.Contains(out, "--wp--block-button--width: 50;") {
		t.Fatalf("missing width CSS var: %q", out)
	}
}

func TestRenderButtonEmptyReturnsEmpty(t *testing.T) {
	src := `<!-- wp:button -->
<div class="wp-block-button"><a class="wp-block-button__link wp-element-button"></a></div>
<!-- /wp:button -->`
	out := blocks.Render(src)
	if strings.TrimSpace(out) != "" {
		t.Fatalf("expected empty for empty button, got %q", out)
	}
}

func TestRenderButtonsSynthesize(t *testing.T) {
	src := `<!-- wp:buttons -->
<!-- wp:button {"text":"One","url":"/a"} /-->
<!-- wp:button {"text":"Two","url":"/b"} /-->
<!-- /wp:buttons -->`
	out := blocks.Render(src)
	if !strings.Contains(out, `class="wp-block-buttons`) {
		t.Fatalf("missing buttons wrapper: %q", out)
	}
	if !strings.Contains(out, "One") || !strings.Contains(out, "Two") {
		t.Fatalf("missing button labels: %q", out)
	}
	if !strings.Contains(out, "wp-block-button__link") {
		t.Fatalf("missing button link class: %q", out)
	}
}

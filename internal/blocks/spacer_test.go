package blocks_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/blocks"
)

func TestRenderSpacerSynthesize(t *testing.T) {
	out := blocks.Render(`<!-- wp:spacer {"height":"48px"} /-->`)
	if !strings.Contains(out, `style="height:48px"`) || !strings.Contains(out, `class="wp-block-spacer"`) || !strings.Contains(out, `aria-hidden="true"`) {
		t.Fatalf("got %q", out)
	}
}

func TestRenderSpacerDefaultHeight(t *testing.T) {
	out := blocks.Render(`<!-- wp:spacer /-->`)
	if !strings.Contains(out, `style="height:100px"`) {
		t.Fatalf("default height: %q", out)
	}
}

func TestRenderSpacerNumericHeight(t *testing.T) {
	out := blocks.Render(`<!-- wp:spacer {"height":72} /-->`)
	if !strings.Contains(out, `style="height:72px"`) {
		t.Fatalf("numeric height: %q", out)
	}
}

func TestRenderSpacerUnitlessString(t *testing.T) {
	out := blocks.Render(`<!-- wp:spacer {"height":"64"} /-->`)
	if !strings.Contains(out, `style="height:64px"`) {
		t.Fatalf("unitless string: %q", out)
	}
}

func TestRenderSpacerWidthOnly(t *testing.T) {
	out := blocks.Render(`<!-- wp:spacer {"width":"40px","height":""} /-->`)
	if !strings.Contains(out, `width:40px`) {
		t.Fatalf("width: %q", out)
	}
	// Empty height should not force the 100px default when width is set.
	if strings.Contains(out, `height:100px`) {
		t.Fatalf("unexpected default height with width: %q", out)
	}
}

func TestRenderSpacerClassName(t *testing.T) {
	out := blocks.Render(`<!-- wp:spacer {"height":"20px","className":"is-style-dots"} /-->`)
	if !strings.Contains(out, `class="wp-block-spacer is-style-dots"`) {
		t.Fatalf("className: %q", out)
	}
}

func TestRenderSpacerPrefersSaved(t *testing.T) {
	out := blocks.Render(`<!-- wp:spacer {"height":"99px"} --><div style="height:12px" aria-hidden="true" class="wp-block-spacer"></div><!-- /wp:spacer -->`)
	if !strings.Contains(out, `height:12px`) || strings.Contains(out, `height:99px`) {
		t.Fatalf("saved markup should win: %q", out)
	}
}

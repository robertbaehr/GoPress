package blocks_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/blocks"
)

// P5.141: horizontal orientation maps length onto width; aria-hidden + margin polish.
func TestRenderSpacerOrientationHorizontalP5141(t *testing.T) {
	out := blocks.Render(`<!-- wp:spacer {"height":"64px","orientation":"horizontal","anchor":"sp-h","className":"flex-gap","style":{"spacing":{"margin":{"top":"8px","bottom":"16px"}}}} /-->`)
	for _, want := range []string{
		`wp-block-spacer`,
		`flex-gap`,
		`id="sp-h"`,
		`aria-hidden="true"`,
		`width:64px`,
		`margin-top:8px`,
		`margin-bottom:16px`,
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in: %q", want, out)
		}
	}
	if strings.Contains(out, `height:64px`) || strings.Contains(out, `height:100px`) {
		t.Fatalf("horizontal spacer should omit height: %q", out)
	}
}

func TestRenderSpacerAriaHiddenOnSavedP5141(t *testing.T) {
	saved := `<!-- wp:spacer {"height":"20px","anchor":"saved-sp"} --><div style="height:12px" class="wp-block-spacer"></div><!-- /wp:spacer -->`
	got := blocks.Render(saved)
	for _, want := range []string{`aria-hidden="true"`, `id="saved-sp"`, `height:12px`} {
		if !strings.Contains(got, want) {
			t.Fatalf("saved polish missing %q in %q", want, got)
		}
	}
}

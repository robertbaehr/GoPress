package wp_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestProcessInteractivityHTMLPassThrough(t *testing.T) {
	in := `<div data-wp-interactive="core/search" data-wp-context='{"isOpen":false}' class="wp-block-search">
<button data-wp-on--click="actions.open">Open</button>
</div>`
	got := wp.ProcessInteractivityHTML(in)
	if got != in {
		t.Fatalf("expected pass-through, got %q", got)
	}
	if !wp.HasInteractivityDirectives(got) {
		t.Fatal("expected data-wp-* detection")
	}
}

func TestKsesPostPreservesDataWpAttrs(t *testing.T) {
	in := `<div data-wp-interactive="core/navigation" class="wp-block-navigation">nav</div>`
	got := wp.KsesPost(in)
	if !strings.Contains(got, `data-wp-interactive="core/navigation"`) {
		t.Fatalf("data-wp-interactive stripped: %q", got)
	}
	if !strings.Contains(got, `class="wp-block-navigation"`) {
		t.Fatalf("class missing: %q", got)
	}
}

package wp_test

import (
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// P8.60: ConstrainDimensions fits inside max box; filter override.
func TestConstrainDimensionsP860(t *testing.T) {
	c := testBootQuery(t)
	w, h := wp.ConstrainDimensions(c, 800, 600, 400, 400)
	if w != 400 || h != 300 {
		t.Fatalf("got %dx%d want 400x300", w, h)
	}
	w, h = wp.ConstrainDimensions(c, 100, 50, 0, 0)
	if w != 100 || h != 50 {
		t.Fatalf("noop %dx%d", w, h)
	}
	c.Hooks.AddFilter("wp_constrain_dimensions", func(value any, _ ...any) any {
		return []int{10, 20}
	}, 10, 5)
	w, h = wp.ConstrainDimensions(c, 800, 600, 400, 400)
	if w != 10 || h != 20 {
		t.Fatalf("filter %dx%d", w, h)
	}
}

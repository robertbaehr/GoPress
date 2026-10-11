package wp_test

import (
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// P9.43: DoAction / ApplyFilters Context helpers.
func TestDoActionApplyFiltersHelpersP943(t *testing.T) {
	c := testBootQuery(t)
	var acted int
	wp.AddAction(c, "p943_action", func(args ...any) {
		if len(args) == 1 && args[0] == "x" {
			acted++
		}
	}, 10, 1)
	wp.DoAction(c, "p943_action", "x")
	if acted != 1 {
		t.Fatalf("acted=%d", acted)
	}
	wp.DoAction(nil, "p943_action", "x")

	wp.AddFilter(c, "p943_filter", func(value any, _ ...any) any {
		if s, ok := value.(string); ok {
			return s + "-ok"
		}
		return value
	}, 10, 1)
	if got := wp.ApplyFilters(c, "p943_filter", "seed"); got != "seed-ok" {
		t.Fatalf("got=%v", got)
	}
	if got := wp.ApplyFilters(nil, "p943_filter", "seed"); got != "seed" {
		t.Fatalf("nil=%v", got)
	}
}

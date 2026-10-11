package wp_test

import (
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// P8.62: ImageMatchesRatio matches same ratio within 1px fuzz.
func TestImageMatchesRatioP862(t *testing.T) {
	if !wp.ImageMatchesRatio(100, 50, 200, 100) {
		t.Fatal("2:1 vs 2:1 should match")
	}
	if !wp.ImageMatchesRatio(100, 50, 101, 50) {
		t.Fatal("1px width fuzz should match when constrained")
	}
	if wp.ImageMatchesRatio(100, 50, 200, 150) {
		t.Fatal("2:1 vs 4:3 should not match")
	}
	if wp.ImageMatchesRatio(0, 50, 100, 50) {
		t.Fatal("zero source width should not match")
	}
	if wp.ImageMatchesRatio(100, 50, 0, 50) {
		t.Fatal("zero target width should not match")
	}
}

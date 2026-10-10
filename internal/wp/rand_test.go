package wp_test

import (
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestRandRangeAndOrder(t *testing.T) {
	// P2.103: wp_rand accepts either order; inclusive bounds.
	for i := 0; i < 40; i++ {
		v := wp.Rand(5, 10)
		if v < 5 || v > 10 {
			t.Fatalf("Rand(5,10)=%d", v)
		}
		v = wp.Rand(10, 5)
		if v < 5 || v > 10 {
			t.Fatalf("Rand(10,5)=%d", v)
		}
	}
	if v := wp.Rand(7, 7); v != 7 {
		t.Fatalf("Rand(7,7)=%d", v)
	}
}

func TestRandDefaultsAndAbsInt(t *testing.T) {
	v := wp.Rand()
	if v < 0 || v > 4294967295 {
		t.Fatalf("Rand()=%d out of default range", v)
	}
	if wp.AbsInt(-3) != 3 || wp.AbsInt(4) != 4 {
		t.Fatal("AbsInt")
	}
	// Single-arg: min set, max default.
	v = wp.Rand(4294967290)
	if v < 4294967290 || v > 4294967295 {
		t.Fatalf("Rand(min)=%d", v)
	}
}

package wp_test

import (
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestHashEquals(t *testing.T) {
	if !wp.HashEquals("abc", "abc") {
		t.Fatal("equal strings")
	}
	if wp.HashEquals("abc", "abd") {
		t.Fatal("differing strings")
	}
	if wp.HashEquals("abc", "ab") {
		t.Fatal("length mismatch")
	}
	if wp.HashEquals("", "x") {
		t.Fatal("empty vs non-empty")
	}
	if !wp.HashEquals("", "") {
		t.Fatal("both empty")
	}
}

package wp_test

import (
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestArrayKeyFirstLastSlice(t *testing.T) {
	in := []string{"a", "b", "c"}
	k, ok := wp.ArrayKeyFirst(in)
	if !ok || k != 0 {
		t.Fatalf("first=%v ok=%v", k, ok)
	}
	k, ok = wp.ArrayKeyLast(in)
	if !ok || k != 2 {
		t.Fatalf("last=%v ok=%v", k, ok)
	}
}

func TestArrayKeyFirstLastMap(t *testing.T) {
	in := map[string]any{"z": 1, "a": 2, "m": 3}
	k, ok := wp.ArrayKeyFirst(in)
	if !ok || k != "a" {
		t.Fatalf("first=%v ok=%v", k, ok)
	}
	k, ok = wp.ArrayKeyLast(in)
	if !ok || k != "z" {
		t.Fatalf("last=%v ok=%v", k, ok)
	}
}

func TestArrayKeyFirstLastEmpty(t *testing.T) {
	if _, ok := wp.ArrayKeyFirst([]any{}); ok {
		t.Fatal("empty slice first")
	}
	if _, ok := wp.ArrayKeyLast(map[string]any{}); ok {
		t.Fatal("empty map last")
	}
	if _, ok := wp.ArrayKeyFirst(nil); ok {
		t.Fatal("nil")
	}
	if _, ok := wp.ArrayKeyFirst("x"); ok {
		t.Fatal("scalar")
	}
}

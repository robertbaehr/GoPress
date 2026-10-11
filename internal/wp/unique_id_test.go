package wp_test

import (
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestUniqueIDIncrements(t *testing.T) {
	wp.ResetUniqueIDsForTest()
	if got := wp.UniqueID(""); got != "1" {
		t.Fatalf("got %q", got)
	}
	if got := wp.UniqueID("id-"); got != "id-2" {
		t.Fatalf("got %q", got)
	}
	if got := wp.UniqueID("id-"); got != "id-3" {
		t.Fatalf("got %q", got)
	}
}

func TestUniquePrefixedIDIndependent(t *testing.T) {
	wp.ResetUniqueIDsForTest()
	if got := wp.UniquePrefixedID("a-"); got != "a-1" {
		t.Fatalf("got %q", got)
	}
	if got := wp.UniquePrefixedID("b-"); got != "b-1" {
		t.Fatalf("got %q", got)
	}
	if got := wp.UniquePrefixedID("a-"); got != "a-2" {
		t.Fatalf("got %q", got)
	}
}

func TestUniqueIDFromValues(t *testing.T) {
	// P2.104: stable hash id from JSON-serialized values.
	a := wp.UniqueIDFromValues(map[string]any{"x": 1, "y": "z"}, "wp-")
	b := wp.UniqueIDFromValues(map[string]any{"x": 1, "y": "z"}, "wp-")
	if a != b {
		t.Fatalf("same input should match: %q vs %q", a, b)
	}
	if len(a) != len("wp-")+8 {
		t.Fatalf("expected prefix+8 hex, got %q", a)
	}
	c := wp.UniqueIDFromValues(map[string]any{"x": 2}, "wp-")
	if a == c {
		t.Fatalf("different input should differ: %q", a)
	}
	empty := wp.UniqueIDFromValues(map[string]any{}, "")
	if len(empty) != 8 {
		t.Fatalf("empty map id=%q", empty)
	}
}

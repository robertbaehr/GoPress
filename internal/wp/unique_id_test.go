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

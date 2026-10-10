package wp_test

import (
	"encoding/json"
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestRecursiveKSortNested(t *testing.T) {
	in := map[string]any{
		"z": 1,
		"a": map[string]any{
			"d": 4,
			"b": 2,
			"c": map[string]any{"y": 1, "x": 0},
		},
		"m": []any{
			map[string]any{"b": 1, "a": 0},
		},
	}
	got := wp.RecursiveKSort(in)
	raw, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"a":{"b":2,"c":{"x":0,"y":1},"d":4},"m":[{"a":0,"b":1}],"z":1}`
	if string(raw) != want {
		t.Fatalf("got %s want %s", raw, want)
	}
	// Input not mutated (still has nested unsorted structure accessible).
	if _, ok := in["z"]; !ok || len(in) != 3 {
		t.Fatalf("input mutated: %#v", in)
	}
}

func TestRecursiveKSortNilAndScalar(t *testing.T) {
	if got := wp.RecursiveKSort(nil); got != nil {
		t.Fatalf("nil: %#v", got)
	}
	if got := wp.RecursiveKSort("x"); got != "x" {
		t.Fatalf("scalar: %#v", got)
	}
	got := wp.RecursiveKSort(map[string]any{})
	raw, _ := json.Marshal(got)
	if string(raw) != "{}" {
		t.Fatalf("empty: %s", raw)
	}
}

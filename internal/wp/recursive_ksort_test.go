package wp_test

import (
	"encoding/json"
	"reflect"
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

func TestRecursiveKSortStringMapAndAnyMap(t *testing.T) {
	// P2.92: map[string]string.
	got := wp.RecursiveKSort(map[string]string{"z": "3", "a": "1", "m": "2"})
	raw, _ := json.Marshal(got)
	if string(raw) != `{"a":"1","m":"2","z":"3"}` {
		t.Fatalf("string map: %s", raw)
	}

	got = wp.RecursiveKSort(map[any]any{"b": 1, "a": 0, 3: "three"})
	m, ok := got.(map[string]any)
	if !ok || m["a"] != 0 || m["b"] != 1 || m["3"] != "three" {
		t.Fatalf("map[any]any: %#v", got)
	}
}

func TestRecursiveKSortInPlace(t *testing.T) {
	in := map[string]any{
		"z": 1,
		"a": map[string]any{"d": 4, "b": 2},
	}
	wp.RecursiveKSortInPlace(&in)
	raw, _ := json.Marshal(in)
	want := `{"a":{"b":2,"d":4},"z":1}`
	if string(raw) != want {
		t.Fatalf("inplace: %s want %s", raw, want)
	}
	// Nested value should still be map[string]any.
	nested, ok := in["a"].(map[string]any)
	if !ok || !reflect.DeepEqual(nested, map[string]any{"b": 2, "d": 4}) {
		t.Fatalf("nested: %#v", in["a"])
	}
	wp.RecursiveKSortInPlace(nil)
	empty := map[string]any(nil)
	wp.RecursiveKSortInPlace(&empty)
}

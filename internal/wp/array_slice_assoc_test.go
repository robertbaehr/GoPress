package wp_test

import (
	"reflect"
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestArraySliceAssocMap(t *testing.T) {
	in := map[string]any{
		"a": 1,
		"b": "two",
		"c": true,
		"d": 4,
	}
	got := wp.ArraySliceAssoc(in, []string{"b", "a", "missing", "c"})
	want := map[string]any{"b": "two", "a": 1, "c": true}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v want %#v", got, want)
	}
	// Does not mutate input.
	if len(in) != 4 {
		t.Fatalf("input mutated: %#v", in)
	}
}

func TestArraySliceAssocStruct(t *testing.T) {
	type row struct {
		Name string
		Age  int
		Skip string
	}
	got := wp.ArraySliceAssoc(row{Name: "Ada", Age: 36, Skip: "x"}, []string{"Name", "Age", "Nope"})
	if got["Name"] != "Ada" || got["Age"] != 36 {
		t.Fatalf("%#v", got)
	}
	if _, ok := got["Nope"]; ok {
		t.Fatal("unexpected Nope")
	}
}

func TestArraySliceAssocEmpty(t *testing.T) {
	if got := wp.ArraySliceAssoc(nil, []string{"a"}); len(got) != 0 {
		t.Fatalf("%#v", got)
	}
	if got := wp.ArraySliceAssoc(map[string]any{"a": 1}, nil); len(got) != 0 {
		t.Fatalf("%#v", got)
	}
	if got := wp.ArraySliceAssoc("not-a-map", []string{"a"}); len(got) != 0 {
		t.Fatalf("%#v", got)
	}
}

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

func TestArraySliceAssocJSONTagAndPointer(t *testing.T) {
	type row struct {
		DisplayName string `json:"name"`
		Age         int    `json:"age"`
		Hidden      string `json:"-"`
	}
	in := &row{DisplayName: "Ada", Age: 36, Hidden: "secret"}
	got := wp.ArraySliceAssoc(in, []string{"name", "age", "-", "missing"})
	if got["name"] != "Ada" || got["age"] != 36 {
		t.Fatalf("%#v", got)
	}
	if _, ok := got["-"]; ok {
		t.Fatal("json:\"-\" must not match key \"-\"")
	}
	// Go field name still works for exported fields.
	got = wp.ArraySliceAssoc(in, []string{"Hidden"})
	if got["Hidden"] != "secret" {
		t.Fatalf("expected Hidden by Go name: %#v", got)
	}
}

func TestArraySliceAssocStringMapAndNil(t *testing.T) {
	in := map[string]string{"a": "1", "b": "2"}
	got := wp.ArraySliceAssoc(in, []string{"b", "a"})
	if got["a"] != "1" || got["b"] != "2" {
		t.Fatalf("%#v", got)
	}
	anyMap := map[string]any{"x": nil, "y": 1}
	got = wp.ArraySliceAssoc(anyMap, []string{"x", "y"})
	if _, ok := got["x"]; !ok {
		t.Fatal("expected nil value for present key x")
	}
	if got["y"] != 1 {
		t.Fatalf("%#v", got)
	}
}

func TestArraySliceAssocIntKeys(t *testing.T) {
	in := map[int]any{0: "zero", 12: "twelve", 3: nil}
	got := wp.ArraySliceAssoc(in, []string{"0", "12", "3", "9"})
	if got["0"] != "zero" || got["12"] != "twelve" {
		t.Fatalf("%#v", got)
	}
	if _, ok := got["3"]; !ok {
		t.Fatal("expected nil for key 3")
	}
	if _, ok := got["9"]; ok {
		t.Fatal("unexpected 9")
	}
	anyKeys := map[any]any{1: "one", "two": 2}
	got = wp.ArraySliceAssoc(anyKeys, []string{"1", "two"})
	if got["1"] != "one" || got["two"] != 2 {
		t.Fatalf("%#v", got)
	}
}

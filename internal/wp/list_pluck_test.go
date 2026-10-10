package wp_test

import (
	"reflect"
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestListPluckMaps(t *testing.T) {
	in := []map[string]any{
		{"id": 1, "name": "a"},
		{"id": 2, "name": "b"},
		{"id": 3, "name": "c"},
	}
	got := wp.ListPluck(in, "name", "")
	want := []any{"a", "b", "c"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("pluck name: %#v want %#v", got, want)
	}

	keyed := wp.ListPluck(in, "name", "id")
	km, ok := keyed.(map[any]any)
	if !ok {
		t.Fatalf("expected map, got %T", keyed)
	}
	if km[1] != "a" || km[2] != "b" || km[3] != "c" {
		t.Fatalf("keyed=%#v", km)
	}
}

func TestListPluckStructs(t *testing.T) {
	type row struct {
		ID   int
		Name string
	}
	in := []row{{1, "x"}, {2, "y"}}
	got := wp.ListPluck(in, "Name", "")
	want := []any{"x", "y"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("struct pluck: %#v", got)
	}
	keyed := wp.ListPluck(in, "Name", "ID").(map[any]any)
	if keyed[1] != "x" || keyed[2] != "y" {
		t.Fatalf("struct keyed=%#v", keyed)
	}
}

func TestListPluckMissingIndexAppends(t *testing.T) {
	in := []map[string]any{
		{"name": "a", "id": 10},
		{"name": "b"}, // no id
		{"name": "c", "id": 12},
	}
	got := wp.ListPluck(in, "name", "id").(map[any]any)
	if got[10] != "a" || got[12] != "c" {
		t.Fatalf("got=%#v", got)
	}
	if got[0] != "b" {
		t.Fatalf("missing index should append at 0, got=%#v", got)
	}
}

func TestListPluckEmptyAndInvalid(t *testing.T) {
	if got := wp.ListPluck(nil, "x", ""); !reflect.DeepEqual(got, []any{}) {
		t.Fatalf("nil: %#v", got)
	}
	if got := wp.ListPluck("nope", "x", "id"); !reflect.DeepEqual(got, map[any]any{}) {
		t.Fatalf("non-list: %#v", got)
	}
	if got := wp.ListPluck([]map[string]any{}, "x", ""); !reflect.DeepEqual(got, []any{}) {
		t.Fatalf("empty: %#v", got)
	}
}

func TestListPluckJSONTagsAndPointers(t *testing.T) {
	type row struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}
	in := []*row{{ID: 1, Name: "alpha"}, {ID: 2, Name: "beta"}}
	got := wp.ListPluck(in, "name", "")
	want := []any{"alpha", "beta"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("json tag pluck: %#v want %#v", got, want)
	}
	keyed := wp.ListPluck(in, "name", "id").(map[any]any)
	if keyed[1] != "alpha" || keyed[2] != "beta" {
		t.Fatalf("json keyed=%#v", keyed)
	}
}

func TestListPluckStringMaps(t *testing.T) {
	in := []map[string]string{
		{"slug": "go", "label": "Go"},
		{"slug": "php", "label": "PHP"},
	}
	got := wp.ListPluck(in, "label", "")
	want := []any{"Go", "PHP"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("string map pluck: %#v", got)
	}
	keyed := wp.ListPluck(in, "label", "slug").(map[any]any)
	if keyed["go"] != "Go" || keyed["php"] != "PHP" {
		t.Fatalf("string map keyed=%#v", keyed)
	}
}

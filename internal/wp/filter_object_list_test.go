package wp_test

import (
	"reflect"
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestFilterObjectListAND(t *testing.T) {
	in := []map[string]any{
		{"name": "a", "public": true, "id": 1},
		{"name": "b", "public": false, "id": 2},
		{"name": "c", "public": true, "id": 3},
	}
	got := wp.FilterObjectList(in, map[string]any{"public": true}, "and", "")
	out, ok := got.([]any)
	if !ok || len(out) != 2 {
		t.Fatalf("got %#v", got)
	}
	plucked := wp.FilterObjectList(in, map[string]any{"public": true}, "AND", "name")
	if !reflect.DeepEqual(plucked, []any{"a", "c"}) {
		t.Fatalf("pluck=%#v", plucked)
	}
}

func TestFilterObjectListORAndNOT(t *testing.T) {
	in := []map[string]any{
		{"type": "post", "status": "publish"},
		{"type": "page", "status": "draft"},
		{"type": "post", "status": "draft"},
	}
	// type=page OR status=publish → first two rows.
	orNames := wp.FilterObjectList(in, map[string]any{"type": "page", "status": "publish"}, "OR", "type")
	if !reflect.DeepEqual(orNames, []any{"post", "page"}) {
		t.Fatalf("OR got %#v", orNames)
	}

	not := wp.FilterObjectList(in, map[string]any{"status": "draft"}, "NOT", "type")
	if !reflect.DeepEqual(not, []any{"post"}) {
		t.Fatalf("NOT got %#v", not)
	}
}

func TestFilterObjectListStructs(t *testing.T) {
	type row struct {
		Name   string
		Public bool
	}
	in := []row{{"a", true}, {"b", false}}
	got := wp.FilterObjectList(in, map[string]any{"Public": true}, "and", "Name")
	if !reflect.DeepEqual(got, []any{"a"}) {
		t.Fatalf("got %#v", got)
	}
}

func TestListFilterEmptyArgs(t *testing.T) {
	in := []map[string]any{{"a": 1}, {"a": 2}}
	got := wp.ListFilter(in, nil, "AND")
	out, _ := got.([]any)
	if len(out) != 2 {
		t.Fatalf("got %#v", got)
	}
	if got := wp.FilterObjectList("nope", map[string]any{"x": 1}, "AND", ""); !reflect.DeepEqual(got, []any{}) {
		t.Fatalf("invalid input: %#v", got)
	}
}

func TestFilterObjectListEmptyArgsIgnoresOperator(t *testing.T) {
	in := []map[string]any{{"name": "a"}, {"name": "b"}}
	// P2.89: empty args returns all rows even with an invalid operator.
	got := wp.FilterObjectList(in, map[string]any{}, "XOR", "name")
	if !reflect.DeepEqual(got, []any{"a", "b"}) {
		t.Fatalf("empty args+invalid op: %#v", got)
	}
	got = wp.FilterObjectList(in, nil, "nope", "")
	out, _ := got.([]any)
	if len(out) != 2 {
		t.Fatalf("nil args+invalid op: %#v", got)
	}
}

func TestFilterObjectListNilStructFieldIsMissing(t *testing.T) {
	type row struct {
		Name   string
		Status *string
	}
	draft := "draft"
	in := []row{{Name: "a", Status: &draft}, {Name: "b", Status: nil}}
	// isset-style: nil Status does not match AND status=draft.
	got := wp.FilterObjectList(in, map[string]any{"Status": "draft"}, "AND", "Name")
	if !reflect.DeepEqual(got, []any{"a"}) {
		t.Fatalf("got %#v", got)
	}
	// NOT status=draft keeps the nil-Status row (missing ≠ match).
	got = wp.FilterObjectList(in, map[string]any{"Status": "draft"}, "NOT", "Name")
	if !reflect.DeepEqual(got, []any{"b"}) {
		t.Fatalf("NOT got %#v", got)
	}
}

func TestFilterObjectListLooseScalarMatch(t *testing.T) {
	in := []map[string]any{
		{"id": 1, "public": "1"},
		{"id": "2", "public": 0},
	}
	got := wp.FilterObjectList(in, map[string]any{"public": true}, "AND", "id")
	if !reflect.DeepEqual(got, []any{1}) {
		t.Fatalf("loose match: %#v", got)
	}
	got = wp.FilterObjectList(in, map[string]any{"id": 2}, "AND", "id")
	if !reflect.DeepEqual(got, []any{"2"}) {
		t.Fatalf("string/int id: %#v", got)
	}
}

func TestListFilterORNotAndStringMaps(t *testing.T) {
	// P2.90: ListFilter covers OR/NOT and map[string]string rows.
	in := []map[string]string{
		{"type": "post", "status": "publish"},
		{"type": "page", "status": "draft"},
		{"type": "post", "status": "draft"},
	}
	or := wp.ListFilter(in, map[string]any{"type": "page", "status": "publish"}, "OR")
	out, _ := or.([]any)
	if len(out) != 2 {
		t.Fatalf("OR got %#v", or)
	}
	not := wp.ListFilter(in, map[string]any{"status": "draft"}, "NOT")
	out, _ = not.([]any)
	if len(out) != 1 {
		t.Fatalf("NOT got %#v", not)
	}
	// Empty operator on ListFilter defaults to AND (PHP signature default).
	and := wp.ListFilter(in, map[string]any{"type": "post", "status": "publish"}, "")
	out, _ = and.([]any)
	if len(out) != 1 {
		t.Fatalf("default AND got %#v", and)
	}
}

func TestFilterObjectListEmptyOperatorInvalid(t *testing.T) {
	in := []map[string]any{{"name": "a"}}
	// P2.90: FilterObjectList with empty operator + non-empty args → empty (WP_List_Util).
	got := wp.FilterObjectList(in, map[string]any{"name": "a"}, "", "")
	if !reflect.DeepEqual(got, []any{}) {
		t.Fatalf("empty op: %#v", got)
	}
}

func TestListFilterNilMapValue(t *testing.T) {
	// array_key_exists: nil map value is present and matches nil want.
	in := []map[string]any{
		{"name": "a", "meta": nil},
		{"name": "b", "meta": "x"},
	}
	got := wp.ListFilter(in, map[string]any{"meta": nil}, "AND")
	out, _ := got.([]any)
	if len(out) != 1 {
		t.Fatalf("nil match: %#v", got)
	}
}

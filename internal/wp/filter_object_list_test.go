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

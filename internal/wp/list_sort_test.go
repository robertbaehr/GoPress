package wp_test

import (
	"reflect"
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestListSortByNameASC(t *testing.T) {
	in := []map[string]any{
		{"name": "c", "id": 3},
		{"name": "a", "id": 1},
		{"name": "b", "id": 2},
	}
	got := wp.ListSort(in, "name", "ASC", false).([]any)
	names := []string{}
	for _, row := range got {
		names = append(names, row.(map[string]any)["name"].(string))
	}
	if !reflect.DeepEqual(names, []string{"a", "b", "c"}) {
		t.Fatalf("got %v", names)
	}
}

func TestListSortNumericDESC(t *testing.T) {
	in := []map[string]any{
		{"id": 1}, {"id": 10}, {"id": 2},
	}
	got := wp.ListSort(in, map[string]string{"id": "DESC"}, "ASC", false).([]any)
	ids := []any{}
	for _, row := range got {
		ids = append(ids, row.(map[string]any)["id"])
	}
	if !reflect.DeepEqual(ids, []any{10, 2, 1}) {
		t.Fatalf("got %#v", ids)
	}
}

func TestListSortStructsMulti(t *testing.T) {
	type row struct {
		Type string
		ID   int
	}
	in := []row{{"post", 2}, {"page", 1}, {"post", 1}}
	got := wp.ListSort(in, []string{"Type", "ID"}, "ASC", false).([]any)
	out := make([]row, len(got))
	for i, v := range got {
		out[i] = v.(row)
	}
	want := []row{{"page", 1}, {"post", 1}, {"post", 2}}
	if !reflect.DeepEqual(out, want) {
		t.Fatalf("got %#v want %#v", out, want)
	}
}

func TestListSortEmptyAndInvalid(t *testing.T) {
	if got := wp.ListSort(nil, "x", "ASC", false); !reflect.DeepEqual(got, []any{}) {
		t.Fatalf("nil: %#v", got)
	}
	if got := wp.ListSort("nope", "x", "ASC", false); !reflect.DeepEqual(got, []any{}) {
		t.Fatalf("invalid: %#v", got)
	}
	in := []map[string]any{{"a": 1}}
	got := wp.ListSort(in, "", "ASC", false).([]any)
	if len(got) != 1 {
		t.Fatalf("empty orderby: %#v", got)
	}
}

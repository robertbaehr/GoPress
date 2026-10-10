package wp_test

import (
	"reflect"
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestIsNumericArray(t *testing.T) {
	cases := []struct {
		name string
		in   any
		want bool
	}{
		{"nil", nil, false},
		{"string", "hi", false},
		{"int", 3, false},
		{"slice", []any{1, 2, 3, 4}, true},
		{"string slice", []string{"a", "b"}, true},
		{"empty slice", []any{}, true},
		{"assoc map", map[string]any{"name": "WordPress"}, false},
		{"empty string map", map[string]any{}, true},
		{"int map contiguous", map[int]any{0: 1, 1: 2}, true},
		{"int map sparse", map[int]any{2: 8, 3: 9}, true},
		{"mixed key map", map[any]any{0: "a", "name": "b"}, false},
		{"digit string key", map[any]any{"0": "a", "1": "b"}, false},
		{"uint map", map[uint]any{0: "x", 2: "y"}, true},
		{"typed nil slice ptr", (*[]any)(nil), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := wp.IsNumericArray(tc.in)
			if got != tc.want {
				t.Fatalf("IsNumericArray(%#v)=%v want %v", tc.in, got, tc.want)
			}
		})
	}
}

func TestParseList(t *testing.T) {
	cases := []struct {
		name string
		in   any
		want []string
	}{
		{"csv", "a,b,c", []string{"a", "b", "c"}},
		{"spaces", "a b  c", []string{"a", "b", "c"}},
		{"mixed", "a, b\tc", []string{"a", "b", "c"}},
		{"int", 42, []string{"42"}},
		{"empty", "", []string{}},
		{"bool true", true, []string{"1"}},
		{"bool false", false, []string{}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := wp.ParseList(tc.in)
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("ParseList(%#v)=%#v want %#v", tc.in, got, tc.want)
			}
		})
	}
}

func TestRestIsArray(t *testing.T) {
	if !wp.RestIsArray("1,2,3") {
		t.Fatal("csv scalar should be array-like")
	}
	if !wp.RestIsArray([]any{1, 2}) {
		t.Fatal("slice should be array-like")
	}
	if wp.RestIsArray(map[string]any{"a": 1}) {
		t.Fatal("assoc map should not be array-like")
	}
	if wp.RestIsArray(nil) {
		t.Fatal("nil should not be array-like")
	}
}

func TestRestSanitizeArray(t *testing.T) {
	got := wp.RestSanitizeArray("a, b")
	want := []any{"a", "b"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v want %#v", got, want)
	}
	got = wp.RestSanitizeArray(map[string]any{"x": 1, "y": 2})
	if len(got) != 2 {
		t.Fatalf("assoc map values len=%d", len(got))
	}
	got = wp.RestSanitizeArray(struct{}{})
	if len(got) != 0 {
		t.Fatalf("struct should sanitize to empty, got %#v", got)
	}
}

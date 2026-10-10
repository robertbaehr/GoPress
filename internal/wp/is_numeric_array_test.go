package wp_test

import (
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

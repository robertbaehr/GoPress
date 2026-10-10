package wp_test

import (
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestValidateBoolean(t *testing.T) {
	cases := []struct {
		in   any
		want bool
	}{
		{true, true},
		{false, false},
		{nil, false},
		{"false", false},
		{"FALSE", false},
		{"False", false},
		{"true", true},
		{"", false},
		{"0", false},
		{"1", true},
		{"yes", true},
		{0, false},
		{1, true},
		{int64(0), false},
		{0.0, false},
		{1.5, true},
		{[]any{}, false},
		{[]any{"x"}, true},
		{map[string]any{}, false},
		{map[string]any{"a": 1}, true},
		{[]byte("false"), false},
		{[]byte("ok"), true},
	}
	for _, tc := range cases {
		got := wp.ValidateBoolean(tc.in)
		if got != tc.want {
			t.Fatalf("ValidateBoolean(%#v)=%v want %v", tc.in, got, tc.want)
		}
	}
}

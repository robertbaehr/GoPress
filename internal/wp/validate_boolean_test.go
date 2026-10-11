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
		{2, true},
		{"0.0", true}, // PHP (bool)"0.0" is true
	}
	for _, tc := range cases {
		got := wp.ValidateBoolean(tc.in)
		if got != tc.want {
			t.Fatalf("ValidateBoolean(%#v)=%v want %v", tc.in, got, tc.want)
		}
	}
}

func TestRestIsBoolean(t *testing.T) {
	cases := []struct {
		in   any
		want bool
	}{
		{true, true},
		{false, true},
		{"true", true},
		{"FALSE", true},
		{"0", true},
		{"1", true},
		{"yes", false},
		{"", false},
		{0, true},
		{1, true},
		{2, false},
		{int64(0), true},
		{1.0, true},
		{1.5, false},
		{nil, false},
		{[]byte("true"), true},
		{[]byte("maybe"), false},
	}
	for _, tc := range cases {
		got := wp.RestIsBoolean(tc.in)
		if got != tc.want {
			t.Fatalf("RestIsBoolean(%#v)=%v want %v", tc.in, got, tc.want)
		}
	}
}

func TestRestSanitizeBoolean(t *testing.T) {
	cases := []struct {
		in   any
		want bool
	}{
		{true, true},
		{false, false},
		{"false", false},
		{"FALSE", false},
		{"0", false},
		{"1", true},
		{"true", true},
		{"yes", true},
		{"", false},
		{0, false},
		{1, true},
		{2, true},
		{[]byte("0"), false},
		{[]byte("false"), false},
		{[]byte("x"), true},
	}
	for _, tc := range cases {
		got := wp.RestSanitizeBoolean(tc.in)
		if got != tc.want {
			t.Fatalf("RestSanitizeBoolean(%#v)=%v want %v", tc.in, got, tc.want)
		}
	}
}

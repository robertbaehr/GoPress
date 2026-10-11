package wp_test

import (
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestIsSerializedBasics(t *testing.T) {
	cases := []struct {
		in     any
		strict bool
		want   bool
	}{
		{nil, true, false},
		{123, true, false},
		{"", true, false},
		{"N;", true, true},
		{"b:1;", true, true},
		{"b:0;", true, true},
		{"i:42;", true, true},
		{"d:1.5;", true, true},
		{`s:5:"hello";`, true, true},
		{`a:1:{i:0;s:1:"a";}`, true, true},
		{"not-serialized", true, false},
		{"s:5:\"hello\"", true, false},  // missing trailing ;
		{"s:5:\"hello\"", false, false}, // loose still requires ; or }
		{`s:5:"hello";x`, false, true},  // loose: trailing junk after ;
		{`O:8:"stdClass":0:{}`, true, true},
		{`E:0:{}`, true, true},
	}
	for _, tc := range cases {
		got := wp.IsSerialized(tc.in, tc.strict)
		if got != tc.want {
			t.Fatalf("IsSerialized(%q, %v)=%v want %v", tc.in, tc.strict, got, tc.want)
		}
	}
}

func TestIsSerializedString(t *testing.T) {
	if !wp.IsSerializedString(`s:3:"foo";`) {
		t.Fatal("expected serialized string")
	}
	if wp.IsSerializedString(`i:1;`) {
		t.Fatal("int is not serialized string")
	}
	if wp.IsSerializedString(42) {
		t.Fatal("non-string")
	}
}

func TestMaybeUnserialize(t *testing.T) {
	got := wp.MaybeUnserialize(`s:5:"hello";`)
	if got != "hello" {
		t.Fatalf("got %#v", got)
	}
	if wp.MaybeUnserialize("plain") != "plain" {
		t.Fatal("plain passthrough")
	}
	if wp.MaybeUnserialize(7) != 7 {
		t.Fatal("non-string passthrough")
	}
	arr := wp.MaybeUnserialize(`a:1:{i:0;i:9;}`)
	m, ok := arr.(map[any]any)
	if !ok {
		t.Fatalf("expected map, got %#v", arr)
	}
	if m[int64(0)] != int64(9) && m[0] != int64(9) && m[0] != 9 {
		// Accept int or int64 keys depending on unserialize.
		if v, ok := m[0]; ok {
			t.Logf("key 0 = %#v", v)
		} else if v, ok := m[int64(0)]; ok {
			if v != int64(9) && v != 9 {
				t.Fatalf("bad value %#v", v)
			}
		} else {
			t.Fatalf("missing key: %#v", m)
		}
	}
}

func TestMaybeSerialize(t *testing.T) {
	out := wp.MaybeSerialize(map[string]any{"a": "b"})
	s, ok := out.(string)
	if !ok || !wp.IsSerialized(s, true) {
		t.Fatalf("expected serialized map: %#v", out)
	}
	// Already-serialized string is double-serialized (WP compat).
	double := wp.MaybeSerialize(`s:3:"foo";`)
	ds, ok := double.(string)
	if !ok || ds == `s:3:"foo";` {
		t.Fatalf("expected double-serialize: %#v", double)
	}
	if !wp.IsSerialized(ds, true) {
		t.Fatalf("double should still look serialized: %q", ds)
	}
	if wp.MaybeSerialize("hello") != "hello" {
		t.Fatal("plain string passthrough")
	}
	if wp.MaybeSerialize(42) != 42 {
		t.Fatal("int passthrough")
	}
}

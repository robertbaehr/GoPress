package wp_test

import (
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestParseStrFlat(t *testing.T) {
	got := wp.ParseStr("foo=bar&num=7")
	if got["foo"] != "bar" || got["num"] != "7" {
		t.Fatalf("got=%v", got)
	}
}

func TestParseStrNested(t *testing.T) {
	got := wp.ParseStr("a[b]=1&a[c]=2")
	m, ok := got["a"].(map[string]any)
	if !ok {
		t.Fatalf("a not map: %v", got)
	}
	if m["b"] != "1" || m["c"] != "2" {
		t.Fatalf("nested=%v", m)
	}
}

func TestParseStrDeepNested(t *testing.T) {
	got := wp.ParseStr("user[name][first]=Ada")
	user, _ := got["user"].(map[string]any)
	name, _ := user["name"].(map[string]any)
	if name["first"] != "Ada" {
		t.Fatalf("got=%v", got)
	}
}

func TestParseStrEmptyBrackets(t *testing.T) {
	got := wp.ParseStr("tag[]=a&tag[]=b")
	m, ok := got["tag"].(map[string]any)
	if !ok {
		t.Fatalf("tag=%v", got["tag"])
	}
	if m["0"] != "a" || m["1"] != "b" {
		t.Fatalf("tag map=%v", m)
	}
}

func TestParseStrRepeatedFlat(t *testing.T) {
	got := wp.ParseStr("x=1&x=2")
	arr, ok := got["x"].([]any)
	if !ok || len(arr) != 2 || arr[0] != "1" || arr[1] != "2" {
		t.Fatalf("x=%v", got["x"])
	}
}

func TestParseStrEmpty(t *testing.T) {
	got := wp.ParseStr("")
	if len(got) != 0 {
		t.Fatalf("got=%v", got)
	}
}

func TestParseStrFiltered(t *testing.T) {
	c := &wp.Context{Hooks: wp.NewHooks()}
	c.Hooks.AddFilter("wp_parse_str", func(value any, args ...any) any {
		m, _ := value.(map[string]any)
		m["injected"] = true
		return m
	}, 10, 1)
	got := c.ParseStrFiltered("a=1")
	if got["a"] != "1" || got["injected"] != true {
		t.Fatalf("got=%v", got)
	}
}

func TestParseStrDotSpaceToUnderscore(t *testing.T) {
	// P2.105: PHP parse_str converts '.' and spaces in keys to '_'.
	got := wp.ParseStr("foo.bar=1&a+b=2&x%20y=3")
	if got["foo_bar"] != "1" {
		t.Fatalf("foo.bar → foo_bar: %v", got)
	}
	if got["a_b"] != "2" {
		t.Fatalf("a+b → a_b: %v", got)
	}
	if got["x_y"] != "3" {
		t.Fatalf("x%%20y → x_y: %v", got)
	}
}

func TestParseStrNestedKeyDot(t *testing.T) {
	got := wp.ParseStr("user[first.name]=Ada")
	user, _ := got["user"].(map[string]any)
	if user["first_name"] != "Ada" {
		t.Fatalf("nested key normalize: %v", got)
	}
}

func TestParseStrAmpEntityAndNUL(t *testing.T) {
	got := wp.ParseStr("a=1&amp;b=2\x00&c=3")
	if got["a"] != "1" || got["b"] != "2" || got["c"] != "3" {
		t.Fatalf("amp/NUL: %v", got)
	}
}

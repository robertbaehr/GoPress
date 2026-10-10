package wp_test

import (
	"net/url"
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestParseArgsMergesDefaults(t *testing.T) {
	got := wp.ParseArgs(map[string]any{"b": 2, "c": 3}, map[string]any{"a": 1, "b": 0})
	if got["a"] != 1 || got["b"] != 2 || got["c"] != 3 {
		t.Fatalf("got=%v", got)
	}
}

func TestParseArgsQueryString(t *testing.T) {
	got := wp.ParseArgs("foo=bar&num=7", map[string]any{"foo": "default", "extra": true})
	if got["foo"] != "bar" || got["num"] != "7" || got["extra"] != true {
		t.Fatalf("got=%v", got)
	}
}

func TestParseArgsNilUsesDefaults(t *testing.T) {
	got := wp.ParseArgs(nil, map[string]any{"x": "y"})
	if len(got) != 1 || got["x"] != "y" {
		t.Fatalf("got=%v", got)
	}
}

func TestParseArgsEmptyDefaults(t *testing.T) {
	got := wp.ParseArgs(map[string]string{"a": "1"}, nil)
	if got["a"] != "1" {
		t.Fatalf("got=%v", got)
	}
}

func TestParseArgsURLValues(t *testing.T) {
	v := url.Values{"name": {"Ada"}, "tag": {"a", "b"}}
	got := wp.ParseArgs(v, map[string]any{"name": ""})
	if got["name"] != "Ada" {
		t.Fatalf("name=%v", got["name"])
	}
	tags, ok := got["tag"].([]any)
	if !ok || len(tags) != 2 || tags[0] != "a" || tags[1] != "b" {
		t.Fatalf("tag=%v", got["tag"])
	}
}

func TestParseArgsDoesNotMutateDefaults(t *testing.T) {
	defaults := map[string]any{"a": 1}
	_ = wp.ParseArgs(map[string]any{"a": 2}, defaults)
	if defaults["a"] != 1 {
		t.Fatalf("defaults mutated: %v", defaults)
	}
}

func TestParseArgsStructObject(t *testing.T) {
	type opts struct {
		PerPage int    `json:"per_page"`
		Order   string `json:"order"`
		hidden  string
	}
	got := wp.ParseArgs(opts{PerPage: 5, Order: "asc", hidden: "x"}, map[string]any{
		"per_page": 10,
		"order":    "desc",
		"status":   "publish",
	})
	if got["per_page"] != 5 || got["order"] != "asc" || got["status"] != "publish" {
		t.Fatalf("got=%v", got)
	}
	if _, ok := got["hidden"]; ok {
		t.Fatalf("unexported field leaked: %v", got)
	}
}

func TestParseArgsStructPointer(t *testing.T) {
	type opts struct {
		Name string `json:"name"`
	}
	got := wp.ParseArgs(&opts{Name: "Ada"}, map[string]any{"name": ""})
	if got["name"] != "Ada" {
		t.Fatalf("got=%v", got)
	}
	gotNil := wp.ParseArgs((*opts)(nil), map[string]any{"name": "default"})
	if gotNil["name"] != "default" {
		t.Fatalf("nil pointer should use defaults: %v", gotNil)
	}
}

func TestParseArgsMapAnyAny(t *testing.T) {
	got := wp.ParseArgs(map[any]any{"a": 1, 2: "two"}, map[string]any{"a": 0, "b": true})
	if got["a"] != 1 || got["2"] != "two" || got["b"] != true {
		t.Fatalf("got=%v", got)
	}
}

func TestParseArgsEmptyQueryUsesDefaults(t *testing.T) {
	got := wp.ParseArgs("", map[string]any{"x": 1})
	if got["x"] != 1 || len(got) != 1 {
		t.Fatalf("got=%v", got)
	}
}

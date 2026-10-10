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

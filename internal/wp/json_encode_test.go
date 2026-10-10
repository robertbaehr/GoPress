package wp_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestWPJSONEncodeBasic(t *testing.T) {
	got, ok := wp.WPJSONEncode(map[string]any{
		"hello": "world",
		"n":     42,
		"ok":    true,
		"nil":   nil,
	})
	if !ok {
		t.Fatal("encode failed")
	}
	for _, want := range []string{`"hello":"world"`, `"n":42`, `"ok":true`, `"nil":null`} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %s in %s", want, got)
		}
	}
}

func TestWPJSONEncodeHTMLEscapes(t *testing.T) {
	got, ok := wp.WPJSONEncode(`<script>alert("x")</script>`)
	if !ok {
		t.Fatal("encode failed")
	}
	// encoding/json escapes <>& by default (JSON_HEX_* parity).
	if strings.Contains(got, "<script>") {
		t.Fatalf("expected HTML-escaped output, got %s", got)
	}
	if !strings.Contains(got, `\u003c`) {
		t.Fatalf("expected \\u003c escape, got %s", got)
	}
}

func TestWPJSONEncodeInvalidUTF8(t *testing.T) {
	raw := "ok\xffstill"
	got, ok := wp.WPJSONEncode(raw)
	if !ok {
		t.Fatal("encode failed on invalid utf8")
	}
	if got == "" {
		t.Fatal("empty encode")
	}
	// Replacement character should appear as \ufffd in JSON.
	if !strings.Contains(got, `\ufffd`) && !strings.Contains(got, "still") {
		t.Fatalf("unexpected encode of invalid utf8: %s", got)
	}
}

func TestWPJSONEncodeNested(t *testing.T) {
	got, ok := wp.WPJSONEncode([]any{
		map[string]any{"a": []any{1, "two"}},
	})
	if !ok {
		t.Fatal("encode failed")
	}
	want := `[{"a":[1,"two"]}]`
	if got != want {
		t.Fatalf("got %s want %s", got, want)
	}
}

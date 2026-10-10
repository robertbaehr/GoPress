package wp_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

type formattingVector struct {
	Input             string `json:"input"`
	EscHTML           string `json:"esc_html"`
	EscAttr           string `json:"esc_attr"`
	EscURL            string `json:"esc_url"`
	EscJS             string `json:"esc_js"`
	SanitizeTextField string `json:"sanitize_text_field"`
	SanitizeKey       string `json:"sanitize_key"`
	SanitizeTitle     string `json:"sanitize_title"`
	WPStripAllTags    string `json:"wp_strip_all_tags"`
}

type formattingCorpus struct {
	Vectors []formattingVector `json:"vectors"`
}

func loadFormattingVectors(t *testing.T) []formattingVector {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	// internal/wp → repo root → tests/golden/formatting/vectors.json
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
	path := filepath.Join(root, "tests", "golden", "formatting", "vectors.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}
	var corpus formattingCorpus
	if err := json.Unmarshal(data, &corpus); err != nil {
		t.Fatalf("json: %v", err)
	}
	if len(corpus.Vectors) == 0 {
		t.Fatal("no vectors")
	}
	return corpus.Vectors
}

// Implemented golden keys and how to compute them.
var goldenFormatters = map[string]func(string) string{
	"esc_html":            wp.EscHTML,
	"esc_attr":            wp.EscAttr,
	"esc_url":             wp.EscURL,
	"esc_js":              wp.EscJS,
	"sanitize_text_field": wp.SanitizeTextField,
	"sanitize_key":        wp.SanitizeKey,
	"sanitize_title":      wp.SanitizeTitle,
	"wp_strip_all_tags":   func(s string) string { return wp.WPStripAllTags(s, false) },
}

func TestFormattingGoldenVectors(t *testing.T) {
	vectors := loadFormattingVectors(t)
	type fail struct {
		i, key string
		input  string
		want   string
		got    string
	}
	var fails []fail
	matched := 0
	total := 0

	for i, v := range vectors {
		wantByKey := map[string]string{
			"esc_html":            v.EscHTML,
			"esc_attr":            v.EscAttr,
			"esc_url":             v.EscURL,
			"esc_js":              v.EscJS,
			"sanitize_text_field": v.SanitizeTextField,
			"sanitize_key":        v.SanitizeKey,
			"sanitize_title":      v.SanitizeTitle,
			"wp_strip_all_tags":   v.WPStripAllTags,
		}
		for key, fn := range goldenFormatters {
			total++
			got := fn(v.Input)
			want := wantByKey[key]
			if got == want {
				matched++
				continue
			}
			fails = append(fails, fail{
				i:     strconv.Itoa(i),
				key:   key,
				input: v.Input,
				want:  want,
				got:   got,
			})
		}
	}

	t.Logf("golden formatting: %d/%d vector-cells match across %d functions", matched, total, len(goldenFormatters))
	if len(fails) > 0 {
		limit := 25
		if len(fails) < limit {
			limit = len(fails)
		}
		for _, f := range fails[:limit] {
			t.Errorf("i=%s %s input=%q\n  want %q\n  got  %q", f.i, f.key, f.input, f.want, f.got)
		}
		if len(fails) > limit {
			t.Errorf("... and %d more mismatches", len(fails)-limit)
		}
	}
}

func TestWPTexturizeExtraGolden(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
	path := filepath.Join(root, "tests", "golden", "formatting", "wptexturize.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}
	var corpus struct {
		Vectors []struct {
			Input       string `json:"input"`
			WPTexturize string `json:"wptexturize"`
		} `json:"vectors"`
	}
	if err := json.Unmarshal(data, &corpus); err != nil {
		t.Fatalf("json: %v", err)
	}
	if len(corpus.Vectors) < 10 {
		t.Fatalf("expected ≥10 wptexturize vectors, got %d", len(corpus.Vectors))
	}
	var fails int
	for i, v := range corpus.Vectors {
		got := wp.Texturize(v.Input)
		if got != v.WPTexturize {
			fails++
			t.Errorf("i=%d input=%q\n  want %q\n  got  %q", i, v.Input, v.WPTexturize, got)
		}
	}
	if fails > 0 {
		t.Fatalf("%d/%d wptexturize mismatches", fails, len(corpus.Vectors))
	}
}

func TestSanitizeTitleExtraGolden(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
	path := filepath.Join(root, "tests", "golden", "formatting", "sanitize_title.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}
	var corpus struct {
		Vectors []struct {
			Input         string `json:"input"`
			SanitizeTitle string `json:"sanitize_title"`
		} `json:"vectors"`
	}
	if err := json.Unmarshal(data, &corpus); err != nil {
		t.Fatalf("json: %v", err)
	}
	if len(corpus.Vectors) < 20 {
		t.Fatalf("expected ≥20 sanitize_title vectors, got %d", len(corpus.Vectors))
	}
	var fails int
	for i, v := range corpus.Vectors {
		got := wp.SanitizeTitle(v.Input)
		if got != v.SanitizeTitle {
			fails++
			t.Errorf("i=%d input=%q\n  want %q\n  got  %q", i, v.Input, v.SanitizeTitle, got)
		}
	}
	t.Logf("sanitize_title extra golden: %d/%d match", len(corpus.Vectors)-fails, len(corpus.Vectors))
}

func TestSizeFormatGolden(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
	path := filepath.Join(root, "tests", "golden", "formatting", "vectors.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}
	var corpus struct {
		SizeFormat []struct {
			Bytes  float64 `json:"bytes"`
			Result string  `json:"result"`
		} `json:"size_format"`
	}
	if err := json.Unmarshal(data, &corpus); err != nil {
		t.Fatalf("json: %v", err)
	}
	if len(corpus.SizeFormat) == 0 {
		t.Fatal("no size_format vectors")
	}
	for _, v := range corpus.SizeFormat {
		got := wp.SizeFormat(v.Bytes, 2)
		if got != v.Result {
			t.Fatalf("SizeFormat(%v,2)=%q want %q", v.Bytes, got, v.Result)
		}
	}
}

func TestEscHTMLBasic(t *testing.T) {
	if got := wp.EscHTML(`O'Reilly & Sons`); got != `O&#039;Reilly &amp; Sons` {
		t.Fatalf("got %q", got)
	}
}

func TestSanitizeKeyBasic(t *testing.T) {
	if got := wp.SanitizeKey("Foo_Bar-baz.123"); got != "foo_bar-baz123" {
		t.Fatalf("got %q", got)
	}
}

func TestEscURLRawSkipsDisplayEncoding(t *testing.T) {
	in := "https://example.com/path?a=1&b=2"
	display := wp.EscURL(in)
	raw := wp.EscURLRaw(in)
	if display == "" || raw == "" {
		t.Fatalf("empty: display=%q raw=%q", display, raw)
	}
	if strings.Contains(raw, "&#038;") {
		t.Fatalf("raw should not encode ampersands: %q", raw)
	}
	if !strings.Contains(display, "&#038;") {
		t.Fatalf("display should encode ampersands: %q", display)
	}
	if wp.SanitizeURL(in) != raw {
		t.Fatalf("SanitizeURL alias mismatch: %q vs %q", wp.SanitizeURL(in), raw)
	}
	if wp.EscURLRaw("javascript:alert(1)") != "" {
		t.Fatal("expected empty for bad protocol")
	}
	if got := wp.EscURLRaw("/relative/path"); got != "/relative/path" {
		t.Fatalf("relative: %q", got)
	}
}

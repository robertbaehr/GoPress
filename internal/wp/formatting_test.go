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
	Input               string `json:"input"`
	EscHTML             string `json:"esc_html"`
	EscAttr             string `json:"esc_attr"`
	EscURL              string `json:"esc_url"`
	EscJS               string `json:"esc_js"`
	SanitizeTextField   string `json:"sanitize_text_field"`
	SanitizeKey         string `json:"sanitize_key"`
	SanitizeTitle       string `json:"sanitize_title"`
	SanitizeFileName    string `json:"sanitize_file_name"`
	WPStripAllTags      string `json:"wp_strip_all_tags"`
	NormalizeWhitespace string `json:"normalize_whitespace"`
	WPTrimWords         string `json:"wp_trim_words"`
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
	"esc_html":             wp.EscHTML,
	"esc_attr":             wp.EscAttr,
	"esc_url":              wp.EscURL,
	"esc_js":               wp.EscJS,
	"sanitize_text_field":  wp.SanitizeTextField,
	"sanitize_key":         wp.SanitizeKey,
	"sanitize_title":       wp.SanitizeTitle,
	"sanitize_file_name":   wp.SanitizeFileName,
	"wp_strip_all_tags":    func(s string) string { return wp.WPStripAllTags(s, false) },
	"normalize_whitespace": wp.NormalizeWhitespace,
	"wp_trim_words":        func(s string) string { return wp.TrimWordsEllipsis(s, 5) },
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
			"esc_html":             v.EscHTML,
			"esc_attr":             v.EscAttr,
			"esc_url":              v.EscURL,
			"esc_js":               v.EscJS,
			"sanitize_text_field":  v.SanitizeTextField,
			"sanitize_key":         v.SanitizeKey,
			"sanitize_title":       v.SanitizeTitle,
			"sanitize_file_name":   v.SanitizeFileName,
			"wp_strip_all_tags":    v.WPStripAllTags,
			"normalize_whitespace": v.NormalizeWhitespace,
			"wp_trim_words":        v.WPTrimWords,
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
			Input                   string `json:"input"`
			SanitizeTitle           string `json:"sanitize_title"`
			SanitizeTitleWithDashes string `json:"sanitize_title_with_dashes"`
			SanitizeTitleForQuery   string `json:"sanitize_title_for_query"`
		} `json:"vectors"`
	}
	if err := json.Unmarshal(data, &corpus); err != nil {
		t.Fatalf("json: %v", err)
	}
	if len(corpus.Vectors) < 30 {
		t.Fatalf("expected ≥30 sanitize_title vectors, got %d", len(corpus.Vectors))
	}
	var fails int
	for i, v := range corpus.Vectors {
		got := wp.SanitizeTitle(v.Input)
		if got != v.SanitizeTitle {
			fails++
			t.Errorf("i=%d sanitize_title input=%q\n  want %q\n  got  %q", i, v.Input, v.SanitizeTitle, got)
		}
		// display-context with_dashes (default PHP context).
		if got := wp.SanitizeTitleWithDashes(v.Input, "", "display"); got != v.SanitizeTitleWithDashes {
			fails++
			t.Errorf("i=%d with_dashes input=%q\n  want %q\n  got  %q", i, v.Input, v.SanitizeTitleWithDashes, got)
		}
		// P2.119 extras include sanitize_title_for_query when present.
		if v.SanitizeTitleForQuery != "" {
			if got := wp.SanitizeTitleForQuery(v.Input); got != v.SanitizeTitleForQuery {
				fails++
				t.Errorf("i=%d for_query input=%q\n  want %q\n  got  %q", i, v.Input, v.SanitizeTitleForQuery, got)
			}
		}
	}
	if fails == 0 {
		t.Logf("sanitize_title extra golden: %d/%d match", len(corpus.Vectors), len(corpus.Vectors))
	}
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

func TestSanitizeFileNameExtras(t *testing.T) {
	cases := []struct{ in, want string }{
		{"photo.jpg", "photo.jpg"},
		{"my file.png", "my-file.png"},
		{"evil.php.jpg", "evil.php_.jpg"}, // php is not an allowed intermediate ext
		{"jpg", "unnamed-file.jpg"},
		{"notanext", "notanext"},
	}
	for _, tc := range cases {
		got := wp.SanitizeFileName(tc.in)
		if got != tc.want {
			t.Errorf("SanitizeFileName(%q)=%q want %q", tc.in, got, tc.want)
		}
	}
}

func TestSanitizeTitleDeepen(t *testing.T) {
	cases := map[string]string{
		"Café résumé":         "cafe-resume",
		"Title — with emdash": "title-with-emdash",
		"«quotes»":            "quotes",
		"!!!":                 "",
		"path/to/file":        "path-to-file",
		"query?x=1":           "queryx1",
		"END.":                "end",
		"hash#frag":           "hashfrag",
	}
	for in, want := range cases {
		if got := wp.SanitizeTitle(in); got != want {
			t.Errorf("SanitizeTitle(%q)=%q want %q", in, got, want)
		}
	}
	// P2.119: query context skips remove_accents (accents stay as octets).
	if got := wp.SanitizeTitleForQuery("Café"); got != "caf%c3%a9" {
		t.Fatalf("SanitizeTitleForQuery Café=%q", got)
	}
	// display with_dashes keeps encoded angle quotes; save strips them.
	if got := wp.SanitizeTitleWithDashes("«quotes»", "", "display"); got != "%c2%abquotes%c2%bb" {
		t.Fatalf("with_dashes display=%q", got)
	}
	if got := wp.SanitizeTitleWithDashes("«quotes»", "", "save"); got != "quotes" {
		t.Fatalf("with_dashes save=%q", got)
	}
	// fallback when empty
	if got := wp.SanitizeTitleWithContext(nil, "!!!", "fallback-slug", "save"); got != "fallback-slug" {
		t.Fatalf("fallback=%q", got)
	}
	c := &wp.Context{Hooks: wp.NewHooks()}
	c.Hooks.AddFilter("sanitize_title", func(value any, args ...any) any {
		if s, ok := value.(string); ok && s != "" {
			return s + "-x"
		}
		return value
	}, 10, 3)
	if got := wp.SanitizeTitleFiltered(c, "Hello World", ""); got != "hello-world-x" {
		t.Fatalf("filtered=%q", got)
	}
}

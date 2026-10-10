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

type ksesVector struct {
	I        int    `json:"i"`
	Input    string `json:"input"`
	KsesData string `json:"wp_kses_data"`
	KsesPost string `json:"wp_kses_post"`
}

type ksesCorpus struct {
	Items []ksesVector `json:"items"`
}

func loadKsesVectors(t *testing.T) []ksesVector {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
	path := filepath.Join(root, "tests", "golden", "kses", "vectors.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}
	var corpus ksesCorpus
	if err := json.Unmarshal(data, &corpus); err != nil {
		t.Fatalf("json: %v", err)
	}
	if len(corpus.Items) == 0 {
		t.Fatal("no vectors")
	}
	return corpus.Items
}

func TestKsesGoldenVectors(t *testing.T) {
	items := loadKsesVectors(t)
	type fail struct {
		i, key string
		input  string
		want   string
		got    string
	}
	var fails []fail
	matched, total := 0, 0

	for _, v := range items {
		cases := []struct {
			key  string
			got  string
			want string
		}{
			{"wp_kses_data", wp.KsesData(v.Input), v.KsesData},
			{"wp_kses_post", wp.KsesPost(v.Input), v.KsesPost},
		}
		for _, c := range cases {
			total++
			if c.got == c.want {
				matched++
				continue
			}
			fails = append(fails, fail{
				i: strconv.Itoa(v.I), key: c.key, input: v.Input,
				want: c.want, got: c.got,
			})
		}
	}

	t.Logf("golden kses: %d/%d vector-cells match (%.1f%%)", matched, total, 100*float64(matched)/float64(total))
	if len(fails) > 0 {
		limit := 30
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

func TestKsesSmoke(t *testing.T) {
	if got := wp.KsesPost(`<p onclick="x">hi</p>`); got != `<p>hi</p>` {
		t.Fatalf("post: %q", got)
	}
	if got := wp.KsesData(`<p onclick="x">hi</p>`); got != `hi` {
		t.Fatalf("data: %q", got)
	}
	if got := wp.KsesUserDescription(`<strong>ok</strong><script>x</script>`); !strings.Contains(got, "<strong>") || strings.Contains(got, "<script>") {
		t.Fatalf("user_description: %q", got)
	}
}

func TestKsesAllowedMatchesGolden(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	path := filepath.Join(filepath.Dir(file), "..", "..", "tests", "golden", "kses", "allowed.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}
	var golden struct {
		AllowedHTML struct {
			Data map[string]json.RawMessage `json:"data"`
			Post map[string]json.RawMessage `json:"post"`
		} `json:"allowed_html"`
	}
	if err := json.Unmarshal(raw, &golden); err != nil {
		t.Fatalf("golden json: %v", err)
	}
	post := wp.AllowedHTMLPost()
	data := wp.AllowedHTMLData()
	for tag := range golden.AllowedHTML.Post {
		if _, ok := post[tag]; !ok {
			t.Fatalf("embed post missing golden tag %q", tag)
		}
	}
	for tag := range golden.AllowedHTML.Data {
		if _, ok := data[tag]; !ok {
			t.Fatalf("embed data missing golden tag %q", tag)
		}
	}
}

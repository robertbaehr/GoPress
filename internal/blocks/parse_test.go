package blocks_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/blocks"
)

const fixtureDir = "/opt/gutenberg/test/integration/fixtures/blocks"

func TestParseHeadingFixture(t *testing.T) {
	assertFixtureParse(t, "core__heading__h2")
}

func TestParseCodeFixture(t *testing.T) {
	assertFixtureParse(t, "core__code")
}

func TestParseHTMLFixture(t *testing.T) {
	assertFixtureParse(t, "core__html")
}

func TestParseSeparatorFixture(t *testing.T) {
	assertFixtureParse(t, "core__separator")
}

func TestParseListNested(t *testing.T) {
	doc := readSerialized(t, "core__list__ul")
	got := blocks.Parse(doc)
	if len(got) < 1 {
		t.Fatal("expected at least one block")
	}
	top := got[0]
	if top.BlockName == nil || *top.BlockName != "core/list" {
		t.Fatalf("blockName=%v", top.BlockName)
	}
	if len(top.InnerBlocks) != 6 {
		t.Fatalf("innerBlocks=%d want 6", len(top.InnerBlocks))
	}
	for i, ib := range top.InnerBlocks {
		if ib.BlockName == nil || *ib.BlockName != "core/list-item" {
			t.Fatalf("inner[%d] name=%v", i, ib.BlockName)
		}
		if ib.InnerHTML == "" {
			t.Fatalf("inner[%d] empty innerHTML", i)
		}
	}
	// Parent innerHTML should retain the ul wrapper with inner blocks stripped.
	if !containsAll(top.InnerHTML, "<ul", "</ul>") {
		t.Fatalf("list innerHTML missing ul wrapper: %q", top.InnerHTML)
	}
	nulls := 0
	for _, c := range top.InnerContent {
		if c == nil {
			nulls++
		}
	}
	if nulls != 6 {
		t.Fatalf("innerContent null markers=%d want 6", nulls)
	}
}

func TestParseParagraphAttrs(t *testing.T) {
	doc := readSerialized(t, "core__paragraph__align-right")
	got := blocks.Parse(doc)
	if len(got) < 1 {
		t.Fatal("expected block")
	}
	b := got[0]
	if b.BlockName == nil || *b.BlockName != "core/paragraph" {
		t.Fatalf("name=%v", b.BlockName)
	}
	style, ok := b.Attrs["style"].(map[string]any)
	if !ok {
		t.Fatalf("attrs=%v", b.Attrs)
	}
	typo, ok := style["typography"].(map[string]any)
	if !ok || typo["textAlign"] != "right" {
		t.Fatalf("typography=%v", style["typography"])
	}
}

func TestParseVoidSeparator(t *testing.T) {
	doc := `<!-- wp:separator {"className":"is-style-wide"} /-->`
	got := blocks.Parse(doc)
	if len(got) != 1 {
		t.Fatalf("len=%d", len(got))
	}
	if got[0].BlockName == nil || *got[0].BlockName != "core/separator" {
		t.Fatalf("name=%v", got[0].BlockName)
	}
	if got[0].Attrs["className"] != "is-style-wide" {
		t.Fatalf("attrs=%v", got[0].Attrs)
	}
	if got[0].InnerHTML != "" {
		t.Fatalf("void innerHTML=%q", got[0].InnerHTML)
	}
}

func TestParseFreeform(t *testing.T) {
	doc := `<p>classic</p>`
	got := blocks.Parse(doc)
	if len(got) != 1 || got[0].BlockName != nil {
		t.Fatalf("got=%+v", got)
	}
	if got[0].InnerHTML != doc {
		t.Fatalf("innerHTML=%q", got[0].InnerHTML)
	}
}

func TestHasBlocks(t *testing.T) {
	if !blocks.HasBlocks(`<!-- wp:paragraph --><p>x</p><!-- /wp:paragraph -->`) {
		t.Fatal("expected true")
	}
	if blocks.HasBlocks(`<p>no blocks</p>`) {
		t.Fatal("expected false")
	}
}

func assertFixtureParse(t *testing.T, name string) {
	t.Helper()
	doc := readSerialized(t, name)
	got := blocks.Parse(doc)
	if len(got) < 1 {
		t.Fatal("no blocks")
	}
	// Compare top block against .parsed.json (ignore trailing freeform newline blocks).
	wantPath := filepath.Join(fixtureDir, name+".parsed.json")
	raw, err := os.ReadFile(wantPath)
	if err != nil {
		t.Fatal(err)
	}
	var want []blocks.Block
	if err := json.Unmarshal(raw, &want); err != nil {
		t.Fatal(err)
	}
	if len(want) < 1 {
		t.Fatal("empty want")
	}
	gj, _ := json.Marshal(normalizeBlock(got[0]))
	wj, _ := json.Marshal(normalizeBlock(want[0]))
	if string(gj) != string(wj) {
		t.Fatalf("mismatch\ngot:  %s\nwant: %s", gj, wj)
	}
}

func normalizeBlock(b blocks.Block) blocks.Block {
	if b.Attrs == nil {
		b.Attrs = map[string]any{}
	}
	if b.InnerBlocks == nil {
		b.InnerBlocks = []blocks.Block{}
	}
	if b.InnerContent == nil {
		b.InnerContent = []any{}
	}
	for i := range b.InnerBlocks {
		b.InnerBlocks[i] = normalizeBlock(b.InnerBlocks[i])
	}
	return b
}

func readSerialized(t *testing.T, name string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(fixtureDir, name+".serialized.html"))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func containsAll(s string, parts ...string) bool {
	for _, p := range parts {
		if !strings.Contains(s, p) {
			return false
		}
	}
	return true
}

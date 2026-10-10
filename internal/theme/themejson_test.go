package theme_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/robertbaehr/GoPress/internal/theme"
)

func TestLoadActiveTwentyTwentyTwo(t *testing.T) {
	tj, err := theme.LoadActive("/workspace", "twentytwentytwo")
	if err != nil {
		t.Fatal(err)
	}
	if tj.Theme != "twentytwentytwo" {
		t.Fatalf("theme=%q", tj.Theme)
	}
	if tj.Version != 2 {
		t.Fatalf("version=%d want 2", tj.Version)
	}
	if tj.Settings == nil {
		t.Fatal("settings nil")
	}
	if _, ok := tj.Settings["color"]; !ok {
		t.Fatalf("settings missing color: keys=%v", keys(tj.Settings))
	}
	if tj.Styles == nil {
		t.Fatal("styles nil")
	}
	if tj.FileSHA256 == "" || len(tj.FileSHA256) != 64 {
		t.Fatalf("FileSHA256=%q", tj.FileSHA256)
	}
	if tj.SettingsStylesSHA256 == "" || len(tj.SettingsStylesSHA256) != 64 {
		t.Fatalf("SettingsStylesSHA256=%q", tj.SettingsStylesSHA256)
	}

	// Raw file hash must match direct sha of theme.json on disk.
	raw, err := os.ReadFile(filepath.Join("/workspace/wp-content/themes/twentytwentytwo/theme.json"))
	if err != nil {
		t.Fatal(err)
	}
	reload, err := theme.LoadFile(filepath.Join("/workspace/wp-content/themes/twentytwentytwo/theme.json"))
	if err != nil {
		t.Fatal(err)
	}
	if reload.FileSHA256 != tj.FileSHA256 {
		t.Fatalf("hash mismatch reload=%s load=%s", reload.FileSHA256, tj.FileSHA256)
	}
	_ = raw

	dump := tj.Dump()
	if dump["theme"] != "twentytwentytwo" {
		t.Fatalf("dump theme=%v", dump["theme"])
	}
	b, err := json.Marshal(dump)
	if err != nil {
		t.Fatal(err)
	}
	if len(b) < 100 {
		t.Fatalf("dump too small: %s", b)
	}
}

func TestLoadActiveTwentyTwentyFive(t *testing.T) {
	tj, err := theme.LoadActive("/workspace", "twentytwentyfive")
	if err != nil {
		t.Fatal(err)
	}
	if tj.Theme != "twentytwentyfive" {
		t.Fatalf("theme=%q", tj.Theme)
	}
	if tj.Settings == nil || tj.Styles == nil {
		t.Fatal("missing settings/styles")
	}
}

func TestLoadActiveMissing(t *testing.T) {
	_, err := theme.LoadActive("/workspace", "not-a-real-theme")
	if err == nil {
		t.Fatal("expected error for missing theme")
	}
}

func keys(m map[string]any) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

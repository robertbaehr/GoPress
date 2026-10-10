package theme_test

import (
	"reflect"
	"testing"

	"github.com/robertbaehr/GoPress/internal/theme"
)

func TestMergeSettingsAndStyles(t *testing.T) {
	base := &theme.JSON{
		Theme:   "base",
		Version: 2,
		Settings: map[string]any{
			"color": map[string]any{
				"palette": []any{
					map[string]any{"slug": "primary", "color": "#111"},
				},
				"custom": true,
			},
			"layout": map[string]any{"contentSize": "600px"},
		},
		Styles: map[string]any{
			"color": map[string]any{"background": "#fff"},
		},
	}
	incoming := &theme.JSON{
		Theme:   "custom",
		Version: 3,
		Settings: map[string]any{
			"color": map[string]any{
				"palette": []any{
					map[string]any{"slug": "accent", "color": "#f00"},
				},
			},
			"typography": map[string]any{"fontSize": true},
		},
		Styles: map[string]any{
			"color": map[string]any{"text": "#222"},
		},
	}
	got, err := theme.Merge(base, incoming)
	if err != nil {
		t.Fatal(err)
	}
	if got.Theme != "custom" || got.Version != 3 {
		t.Fatalf("theme=%q version=%d", got.Theme, got.Version)
	}
	if got.FileSHA256 != "" {
		t.Fatalf("FileSHA256 should be cleared: %q", got.FileSHA256)
	}
	if got.SettingsStylesSHA256 == "" {
		t.Fatal("expected recomputed SettingsStylesSHA256")
	}

	color, _ := got.Settings["color"].(map[string]any)
	if color == nil {
		t.Fatalf("settings.color missing: %#v", got.Settings)
	}
	// Leaf array palette replaced (not appended).
	pal, _ := color["palette"].([]any)
	if len(pal) != 1 {
		t.Fatalf("palette=%#v", pal)
	}
	slug, _ := pal[0].(map[string]any)
	if slug["slug"] != "accent" {
		t.Fatalf("palette slug=%v", slug)
	}
	if color["custom"] != true {
		t.Fatalf("expected custom preserved: %#v", color)
	}
	if _, ok := got.Settings["typography"]; !ok {
		t.Fatal("expected typography from incoming")
	}
	layout, _ := got.Settings["layout"].(map[string]any)
	if layout["contentSize"] != "600px" {
		t.Fatalf("layout=%#v", layout)
	}
	stylesColor, _ := got.Styles["color"].(map[string]any)
	if stylesColor["background"] != "#fff" || stylesColor["text"] != "#222" {
		t.Fatalf("styles.color=%#v", stylesColor)
	}

	// Inputs not mutated.
	basePal, _ := base.Settings["color"].(map[string]any)["palette"].([]any)
	if len(basePal) != 1 {
		t.Fatal("base mutated")
	}
}

func TestMergeLayersOrder(t *testing.T) {
	a := &theme.JSON{Version: 1, Settings: map[string]any{"a": 1, "shared": "default"}}
	b := &theme.JSON{Version: 2, Settings: map[string]any{"b": 2, "shared": "theme"}}
	c := &theme.JSON{Version: 3, Settings: map[string]any{"c": 3, "shared": "custom"}}
	got, err := theme.MergeLayers(a, nil, b, c)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]any{"a": 1, "b": 2, "c": 3, "shared": "custom"}
	if !reflect.DeepEqual(got.Settings, want) {
		t.Fatalf("got %#v want %#v", got.Settings, want)
	}
	if got.Version != 3 {
		t.Fatalf("version=%d", got.Version)
	}
}

func TestMergeNilIncomingClonesBase(t *testing.T) {
	base := &theme.JSON{Theme: "t", Version: 2, Settings: map[string]any{"x": 1}}
	got, err := theme.Merge(base, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got.Settings["x"] != 1 || got == base {
		t.Fatalf("clone failed: %#v", got)
	}
	base.Settings["x"] = 9
	if got.Settings["x"] != 1 {
		t.Fatal("clone shared map")
	}
}

package wp_test

import (
	"testing"

	"github.com/robertbaehr/GoPress/internal/theme"
	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestBlockStyleRegistryRegisterGetUnregister(t *testing.T) {
	r := wp.NewBlockStyleRegistry()
	if !r.RegisterBlockStyle("core/separator", wp.BlockStyle{Name: "wide", Label: "Wide Line"}) {
		t.Fatal("register failed")
	}
	if !r.IsRegistered("core/separator", "wide") {
		t.Fatal("expected wide registered")
	}
	got, ok := r.Get("core/separator", "wide")
	if !ok || got.Label != "Wide Line" {
		t.Fatalf("get: %#v ok=%v", got, ok)
	}
	if r.RegisterBlockStyle("core/separator", wp.BlockStyle{Name: "has space"}) {
		t.Fatal("spaces in style name must fail")
	}
	if !r.UnregisterBlockStyle("core/separator", "wide") {
		t.Fatal("unregister failed")
	}
	if r.IsRegistered("core/separator", "wide") {
		t.Fatal("expected unregistered")
	}
}

func TestContextRegisterBlockStyleAndDefaults(t *testing.T) {
	c := bootWP(t)
	if c.BlockStyles == nil {
		t.Fatal("Boot should wire BlockStyles")
	}
	if !c.BlockStyles.IsRegistered("core/separator", "wide") {
		t.Fatal("default separator/wide missing")
	}
	if !c.RegisterBlockStyle("core/group", wp.BlockStyle{Name: "shadow", Label: "Shadow"}) {
		t.Fatal("context register failed")
	}
	if !c.BlockStyles.IsRegistered("core/group", "shadow") {
		t.Fatal("expected shadow on core/group")
	}
	if !c.UnregisterBlockStyle("core/group", "shadow") {
		t.Fatal("context unregister failed")
	}
	all := c.BlockStyles.All()
	if len(all["core/separator"]) < 2 {
		t.Fatalf("expected default separator styles, got %#v", all["core/separator"])
	}
}

func TestRegisterBlockStylesMulti(t *testing.T) {
	r := wp.NewBlockStyleRegistry()
	ok := r.RegisterBlockStyles([]string{"core/quote", "core/pullquote"}, wp.BlockStyle{
		Name:  "plain",
		Label: "Plain",
	})
	if !ok {
		t.Fatal("multi register failed")
	}
	if !r.IsRegistered("core/quote", "plain") || !r.IsRegistered("core/pullquote", "plain") {
		t.Fatal("expected style on both blocks")
	}
}

func TestRegisterBlockStyleVariationsFromThemeJSON(t *testing.T) {
	r := wp.NewBlockStyleRegistry()
	_ = r.RegisterBlockStyle("core/group", wp.BlockStyle{Name: "section-1", Label: "Existing"})
	wp.RegisterBlockStyleVariationsFromThemeJSON(r, []theme.StyleVariation{
		{
			Slug:       "section-1",
			Title:      "Style 1",
			BlockTypes: []string{"core/group", "core/columns"},
			Styles:     map[string]any{"color": map[string]any{"background": "red"}},
		},
		{
			Slug:       "text-display",
			Title:      "Display",
			BlockTypes: []string{"core/heading", "core/paragraph"},
			Styles:     map[string]any{"typography": map[string]any{"fontSize": "2rem"}},
		},
		{
			Slug:       "empty",
			Title:      "Empty",
			BlockTypes: []string{"core/group"},
			// no styles → skipped
		},
	})
	// Existing registration preserved.
	got, ok := r.Get("core/group", "section-1")
	if !ok || got.Label != "Existing" {
		t.Fatalf("existing overwritten: %+v", got)
	}
	if !r.IsRegistered("core/columns", "section-1") {
		t.Fatal("expected section-1 on columns")
	}
	disp, ok := r.Get("core/heading", "text-display")
	if !ok || disp.Label != "Display" || disp.StyleData == nil {
		t.Fatalf("text-display: %+v ok=%v", disp, ok)
	}
	if r.IsRegistered("core/group", "empty") {
		t.Fatal("variation without styles must be skipped")
	}
}

func TestBootRegistersThemeStyleVariations(t *testing.T) {
	c := bootWP(t)
	if c.BlockStyles == nil {
		t.Fatal("nil BlockStyles")
	}
	// Fixture site uses twentytwentyfive which ships section-* and text-display.
	if !c.BlockStyles.IsRegistered("core/group", "section-1") {
		t.Fatal("expected section-1 from theme styles/sections")
	}
	if !c.BlockStyles.IsRegistered("core/heading", "text-display") {
		t.Fatal("expected text-display from theme styles/blocks")
	}
	got, _ := c.BlockStyles.Get("core/group", "section-1")
	if got.StyleData == nil {
		t.Fatal("expected StyleData from theme.json partial")
	}
}

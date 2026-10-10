package wp_test

import (
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestPatternCategoryRegistry(t *testing.T) {
	r := wp.NewPatternCategoryRegistry()
	if !r.RegisterPatternCategory("banner", wp.PatternCategory{Label: "Banners"}) {
		t.Fatal("register failed")
	}
	got, ok := r.Get("banner")
	if !ok || got.Label != "Banners" || got.Name != "banner" {
		t.Fatalf("get=%#v ok=%v", got, ok)
	}
	if !r.UnregisterPatternCategory("banner") {
		t.Fatal("unregister failed")
	}
	if _, ok := r.Get("banner"); ok {
		t.Fatal("expected missing after unregister")
	}
}

func TestRegisterDefaultPatternCategories(t *testing.T) {
	r := wp.NewPatternCategoryRegistry()
	wp.RegisterDefaultPatternCategories(r, "twentytwentyfive")
	all := r.All()
	if len(all) < 23 {
		t.Fatalf("expected core+TT5 categories, got %d", len(all))
	}
	if _, ok := r.Get("featured"); !ok {
		t.Fatal("missing featured")
	}
	if _, ok := r.Get("twentytwentyfive_page"); !ok {
		t.Fatal("missing TT5 page category")
	}
	// Stable sort by name.
	for i := 1; i < len(all); i++ {
		if all[i-1].Name >= all[i].Name {
			t.Fatalf("unsorted: %q then %q", all[i-1].Name, all[i].Name)
		}
	}
}

func TestBootRegistersPatternCategories(t *testing.T) {
	c := bootWP(t)
	if c.PatternCategories == nil {
		t.Fatal("nil PatternCategories")
	}
	if _, ok := c.PatternCategories.Get("query"); !ok {
		t.Fatal("expected core query category after boot")
	}
}

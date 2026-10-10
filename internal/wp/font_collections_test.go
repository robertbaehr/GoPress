package wp_test

import (
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestFontLibraryRegisterGet(t *testing.T) {
	l := wp.NewFontLibrary()
	if !l.RegisterFontCollection("google-fonts", wp.FontCollection{Name: "Google Fonts"}) {
		t.Fatal("register failed")
	}
	col, ok := l.Get("google-fonts")
	if !ok || col.Name != "Google Fonts" || col.Slug != "google-fonts" {
		t.Fatalf("got=%+v ok=%v", col, ok)
	}
	if col.FontFamilies == nil {
		t.Fatal("expected non-nil font_families")
	}
}

func TestRegisterDefaultFontCollections(t *testing.T) {
	l := wp.NewFontLibrary()
	wp.RegisterDefaultFontCollections(l)
	col, ok := l.Get("google-fonts")
	if !ok {
		t.Fatal("missing google-fonts")
	}
	if len(col.Categories) != 5 {
		t.Fatalf("categories=%d", len(col.Categories))
	}
	if len(l.All()) != 1 {
		t.Fatalf("all=%d", len(l.All()))
	}
}

func TestBootRegistersFontCollections(t *testing.T) {
	c := bootWP(t)
	if c.FontLibrary == nil {
		t.Fatal("nil FontLibrary")
	}
	if _, ok := c.FontLibrary.Get("google-fonts"); !ok {
		t.Fatal("expected google-fonts at boot")
	}
}

package wp_test

import (
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestLoadPatternsThemeAndDB(t *testing.T) {
	c := bootWP(t)
	// Boot already loads patterns; re-load to exercise LoadPatterns directly.
	c.Patterns = wp.NewPatternRegistry()
	if err := wp.LoadPatterns(c); err != nil {
		t.Fatal(err)
	}
	all := c.Patterns.All()
	if len(all) < 5 {
		t.Fatalf("expected theme+DB patterns, got %d", len(all))
	}
	themeHit, dbHit := false, false
	for _, p := range all {
		if p.Source == "theme" && p.Name != "" {
			themeHit = true
		}
		if p.Source == "wp_block" {
			dbHit = true
		}
	}
	if !themeHit {
		t.Fatal("missing theme patterns")
	}
	if !dbHit {
		t.Fatal("missing wp_block patterns from DB")
	}
	if _, ok := c.Patterns.Get("twentytwentyfive/hidden-404"); !ok {
		t.Fatal("expected hidden-404 pattern registered")
	}
}

func TestEnsurePatternsIdempotent(t *testing.T) {
	c := bootWP(t)
	if err := wp.EnsurePatterns(c); err != nil {
		t.Fatal(err)
	}
	n := len(c.Patterns.All())
	if err := wp.EnsurePatterns(c); err != nil {
		t.Fatal(err)
	}
	if len(c.Patterns.All()) != n {
		t.Fatalf("EnsurePatterns reloaded: %d -> %d", n, len(c.Patterns.All()))
	}
}

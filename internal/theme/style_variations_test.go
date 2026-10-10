package theme_test

import (
	"testing"

	"github.com/robertbaehr/GoPress/internal/theme"
)

func TestLoadStyleVariationsBlockScopeTwentyTwentyFive(t *testing.T) {
	vars, err := theme.LoadStyleVariations("/workspace", "twentytwentyfive", "block")
	if err != nil {
		t.Fatal(err)
	}
	if len(vars) == 0 {
		t.Fatal("expected block-scoped style variations")
	}
	var foundSection, foundDisplay bool
	for _, v := range vars {
		if len(v.BlockTypes) == 0 || len(v.Styles) == 0 {
			t.Fatalf("block variation missing blockTypes/styles: %+v", v)
		}
		if v.Slug == "section-1" {
			foundSection = true
		}
		if v.Slug == "text-display" {
			foundDisplay = true
		}
	}
	if !foundSection {
		t.Fatal("expected section-1 variation")
	}
	if !foundDisplay {
		t.Fatal("expected text-display variation")
	}
}

func TestLoadStyleVariationsThemeScopeExcludesBlockTypes(t *testing.T) {
	vars, err := theme.LoadStyleVariations("/workspace", "twentytwentyfive", "theme")
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range vars {
		if len(v.BlockTypes) > 0 {
			t.Fatalf("theme scope should omit blockTypes: %+v", v)
		}
	}
	if len(vars) == 0 {
		t.Fatal("expected theme-scoped color/typography variations")
	}
}

func TestLoadStyleVariationsMissingStylesDir(t *testing.T) {
	vars, err := theme.LoadStyleVariations("/workspace", "twentytwentytwo", "block")
	if err != nil {
		t.Fatal(err)
	}
	// twentytwentytwo has no /styles directory.
	if len(vars) != 0 {
		t.Fatalf("want empty, got %d", len(vars))
	}
}

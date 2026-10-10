package blocks_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/blocks"
)

func TestRenderCategoriesList(t *testing.T) {
	env := &blocks.Env{
		ListCategories: func(taxonomy string) []blocks.CategoryItem {
			if taxonomy != "category" {
				t.Fatalf("taxonomy=%q", taxonomy)
			}
			return []blocks.CategoryItem{
				{ID: 1, Name: "News", URL: "/category/news/", Count: 3},
				{ID: 2, Name: "Empty", URL: "/category/empty/", Count: 0},
			}
		},
	}
	raw := `<!-- wp:categories {"showPostCounts":true} /-->`
	out := blocks.RenderWith(env, raw)
	if !strings.Contains(out, `wp-block-categories-list`) {
		t.Fatalf("expected list class: %q", out)
	}
	if !strings.Contains(out, `wp-block-categories-taxonomy-category`) {
		t.Fatalf("expected taxonomy class: %q", out)
	}
	if !strings.Contains(out, `href="/category/news/"`) || !strings.Contains(out, "News") {
		t.Fatalf("expected News link: %q", out)
	}
	if !strings.Contains(out, "(3)") {
		t.Fatalf("expected count: %q", out)
	}
	if strings.Contains(out, "Empty") {
		t.Fatalf("empty terms hidden by default: %q", out)
	}
}

func TestRenderCategoriesHierarchy(t *testing.T) {
	env := &blocks.Env{
		ListCategories: func(string) []blocks.CategoryItem {
			return []blocks.CategoryItem{
				{ID: 1, Name: "Parent", URL: "/category/parent/", Count: 1, Parent: 0},
				{ID: 2, Name: "Child", URL: "/category/child/", Count: 1, Parent: 1},
			}
		},
	}
	raw := `<!-- wp:categories {"showHierarchy":true} /-->`
	out := blocks.RenderWith(env, raw)
	if !strings.Contains(out, `class="children"`) {
		t.Fatalf("expected children ul: %q", out)
	}
	if !strings.Contains(out, "Child") || !strings.Contains(out, "Parent") {
		t.Fatalf("expected parent+child: %q", out)
	}
}

func TestRenderCategoriesShowOnlyTopLevel(t *testing.T) {
	env := &blocks.Env{
		ListCategories: func(string) []blocks.CategoryItem {
			return []blocks.CategoryItem{
				{ID: 1, Name: "Parent", URL: "/category/parent/", Count: 1, Parent: 0},
				{ID: 2, Name: "Child", URL: "/category/child/", Count: 1, Parent: 1},
			}
		},
	}
	raw := `<!-- wp:categories {"showOnlyTopLevel":true} /-->`
	out := blocks.RenderWith(env, raw)
	if strings.Contains(out, "Child") {
		t.Fatalf("child should be omitted: %q", out)
	}
	if !strings.Contains(out, "Parent") {
		t.Fatalf("expected parent: %q", out)
	}
}

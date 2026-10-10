package theme_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/theme"
)

func TestLoadThemePatternsFromWorkspace(t *testing.T) {
	patterns, err := theme.LoadThemePatterns("/workspace/", "twentytwentyfive")
	if err != nil {
		t.Fatal(err)
	}
	if len(patterns) < 5 {
		t.Fatalf("expected several TT5 patterns, got %d", len(patterns))
	}
	found := false
	for _, p := range patterns {
		if p.Slug == "twentytwentyfive/hidden-404" {
			found = true
			if p.Title == "" {
				t.Fatal("missing title")
			}
			if p.Inserter {
				t.Fatal("hidden-404 should have Inserter: no")
			}
			if p.Content == "" {
				t.Fatal("expected content after PHP close tag")
			}
		}
	}
	if !found {
		t.Fatal("missing twentytwentyfive/hidden-404")
	}
}

func TestHiddenBlogHeadingKeepsEscHTMLEString(t *testing.T) {
	patterns, err := theme.LoadThemePatterns("/workspace/", "twentytwentyfive")
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range patterns {
		if p.Slug != "twentytwentyfive/hidden-blog-heading" {
			continue
		}
		if !strings.Contains(p.Content, "Blog") {
			t.Fatalf("expected Blog from esc_html_e, got %q", p.Content)
		}
		return
	}
	t.Fatal("missing twentytwentyfive/hidden-blog-heading")
}

func TestHidden404KeepsEscHTMLXString(t *testing.T) {
	patterns, err := theme.LoadThemePatterns("/workspace/", "twentytwentyfive")
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range patterns {
		if p.Slug != "twentytwentyfive/hidden-404" {
			continue
		}
		if !strings.Contains(p.Content, "Page not found") {
			t.Fatalf("expected Page not found from esc_html_x, got %q", p.Content)
		}
		return
	}
	t.Fatal("missing twentytwentyfive/hidden-404")
}

func TestParsePatternMissingSlugSkipped(t *testing.T) {
	dir := t.TempDir()
	themeDir := filepath.Join(dir, "wp-content", "themes", "demo", "patterns")
	if err := os.MkdirAll(themeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	raw := "<?php\n/**\n * Title: No Slug\n */\n?>\n<p>x</p>\n"
	if err := os.WriteFile(filepath.Join(themeDir, "noslug.php"), []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	patterns, err := theme.LoadThemePatterns(dir, "demo")
	if err != nil {
		t.Fatal(err)
	}
	if len(patterns) != 0 {
		t.Fatalf("want 0 patterns without slug, got %#v", patterns)
	}
}

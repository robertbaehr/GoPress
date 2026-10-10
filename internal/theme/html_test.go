package theme_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/theme"
)

func TestLoadTemplateHTMLIndex(t *testing.T) {
	content, ok := theme.LoadTemplateHTML("/workspace", "twentytwentyfive", "index")
	if !ok {
		t.Fatal("expected index.html")
	}
	if !strings.Contains(content, "wp:template-part") {
		t.Fatalf("unexpected content: %q", content[:min(80, len(content))])
	}
}

func TestLoadTemplateHTMLHome(t *testing.T) {
	content, ok := theme.LoadTemplateHTML("/workspace", "twentytwentytwo", "home")
	if !ok {
		t.Fatal("expected home.html")
	}
	if !strings.Contains(content, "wp:query") {
		t.Fatalf("unexpected content: %q", content[:min(80, len(content))])
	}
}

func TestLoadPartHTMLHeader(t *testing.T) {
	content, ok := theme.LoadPartHTML("/workspace", "twentytwentyfive", "header")
	if !ok {
		t.Fatal("expected header.html")
	}
	if content == "" {
		t.Fatal("empty part")
	}
}

func TestLoadTemplateHTMLMissing(t *testing.T) {
	if _, ok := theme.LoadTemplateHTML("/workspace", "twentytwentyfive", "nope-missing"); ok {
		t.Fatal("expected miss")
	}
}

func TestLoadTemplateHTMLRejectsTraversal(t *testing.T) {
	if _, ok := theme.LoadTemplateHTML("/workspace", "twentytwentyfive", "../index"); ok {
		t.Fatal("traversal should fail")
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

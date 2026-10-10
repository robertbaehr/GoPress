package wp_test

import (
	"context"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/config"
	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestResolveBlockTemplateFromDB(t *testing.T) {
	c := bootWP(t)
	content, ok := wp.ResolveBlockTemplate(c, "home", "index")
	if !ok {
		t.Fatal("expected fixture wp_template index")
	}
	if !strings.Contains(content, "Customized index template") {
		t.Fatalf("want DB customized content, got %q", content)
	}
}

func TestResolveTemplatePartFromDB(t *testing.T) {
	c := bootWP(t)
	content, ok := wp.ResolveTemplatePart(c, "header")
	if !ok {
		t.Fatal("expected fixture wp_template_part header")
	}
	if !strings.Contains(content, "Customized header part") {
		t.Fatalf("got %q", content)
	}
}

func TestResolveBlockTemplateThemeFileFallback(t *testing.T) {
	c := bootWP(t)
	// No DB template named "single" in fixture; theme file should win.
	content, ok := wp.ResolveBlockTemplate(c, "single")
	if !ok {
		t.Fatal("expected theme single.html")
	}
	if !strings.Contains(content, "<!-- wp:") {
		t.Fatalf("expected block markup, got %q", content[:min(60, len(content))])
	}
}

func TestActiveStylesheet(t *testing.T) {
	c := bootWP(t)
	if got := wp.ActiveStylesheet(c); got != "twentytwentyfive" {
		t.Fatalf("stylesheet=%q", got)
	}
}

func bootWP(t *testing.T) *wp.Context {
	t.Helper()
	cfg := &config.Config{
		DBName: "gopress_go", DBUser: "gopress", DBPassword: "gopress",
		DBHost: "127.0.0.1:3306", DBCharset: "utf8mb4", TablePrefix: "wp_",
		ABSPATH: "/workspace/",
		AuthKey: "k", SecureAuthKey: "k", LoggedInKey: "k", NonceKey: "k",
		AuthSalt: "s", SecureAuthSalt: "s", LoggedInSalt: "s", NonceSalt: "s",
	}
	c, err := wp.Boot(context.Background(), cfg)
	if err != nil {
		t.Skipf("Boot unavailable: %v", err)
	}
	t.Cleanup(func() { _ = c.DB.Close() })
	return c
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

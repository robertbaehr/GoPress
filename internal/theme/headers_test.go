package theme_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/robertbaehr/GoPress/internal/theme"
)

func TestGetThemeDataTwentyTwentyFive(t *testing.T) {
	path := filepath.Join("/workspace", "wp-content", "themes", "twentytwentyfive", "style.css")
	if _, err := os.Stat(path); err != nil {
		t.Skip(err)
	}
	data, err := theme.GetThemeData(path)
	if err != nil {
		t.Fatal(err)
	}
	if data.Name != "Twenty Twenty-Five" {
		t.Fatalf("Name=%q", data.Name)
	}
	if data.Author == "" || data.Version == "" || data.Description == "" {
		t.Fatalf("incomplete headers: %+v", data)
	}
}

func TestListInstalledThemes(t *testing.T) {
	themes, err := theme.ListInstalledThemes("/workspace/")
	if err != nil {
		t.Fatal(err)
	}
	if len(themes) < 4 {
		t.Fatalf("expected ≥4 themes, got %d", len(themes))
	}
	found := false
	for _, th := range themes {
		if th.Slug == "twentytwentyfive" && th.Name != "" {
			found = true
		}
	}
	if !found {
		t.Fatalf("missing twentytwentyfive: %+v", themes)
	}
}

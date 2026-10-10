// Package theme — HTML block template / template-part file loading (P5.5+).
package theme

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// LoadTemplateHTML reads templates/{slug}.html from a theme directory.
// Returns ("", false) when the file is missing.
func LoadTemplateHTML(absPath, stylesheet, slug string) (string, bool) {
	return loadThemeHTML(absPath, stylesheet, "templates", slug)
}

// LoadPartHTML reads parts/{slug}.html from a theme directory.
// Returns ("", false) when the file is missing.
func LoadPartHTML(absPath, stylesheet, slug string) (string, bool) {
	return loadThemeHTML(absPath, stylesheet, "parts", slug)
}

func loadThemeHTML(absPath, stylesheet, subdir, slug string) (string, bool) {
	stylesheet = strings.TrimSpace(stylesheet)
	slug = sanitizeSlug(slug)
	if absPath == "" || stylesheet == "" || slug == "" {
		return "", false
	}
	path := filepath.Join(absPath, "wp-content", "themes", stylesheet, subdir, slug+".html")
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", false
	}
	return string(raw), true
}

// ThemeHTMLPath returns the absolute path for a theme HTML file (may not exist).
func ThemeHTMLPath(absPath, stylesheet, subdir, slug string) string {
	stylesheet = strings.TrimSpace(stylesheet)
	slug = sanitizeSlug(slug)
	return filepath.Join(absPath, "wp-content", "themes", stylesheet, subdir, slug+".html")
}

func sanitizeSlug(slug string) string {
	slug = strings.TrimSpace(slug)
	if slug == "" {
		return ""
	}
	// Reject path traversal; WP validate_file subset.
	if strings.Contains(slug, "..") || strings.ContainsAny(slug, `/\`) {
		return ""
	}
	return slug
}

// MustLoadTemplateHTML is like LoadTemplateHTML but returns an error when missing.
func MustLoadTemplateHTML(absPath, stylesheet, slug string) (string, error) {
	content, ok := LoadTemplateHTML(absPath, stylesheet, slug)
	if !ok {
		return "", fmt.Errorf("theme: template %s/%s.html not found", stylesheet, slug)
	}
	return content, nil
}

// HTMLFile is a theme templates/ or parts/ HTML file.
type HTMLFile struct {
	Slug    string
	Content string
}

// ListTemplateHTML lists templates/*.html for a theme (sorted by slug).
func ListTemplateHTML(absPath, stylesheet string) []HTMLFile {
	return listThemeHTML(absPath, stylesheet, "templates")
}

// ListPartHTML lists parts/*.html for a theme (sorted by slug).
func ListPartHTML(absPath, stylesheet string) []HTMLFile {
	return listThemeHTML(absPath, stylesheet, "parts")
}

func listThemeHTML(absPath, stylesheet, subdir string) []HTMLFile {
	stylesheet = strings.TrimSpace(stylesheet)
	if absPath == "" || stylesheet == "" {
		return nil
	}
	dir := filepath.Join(absPath, "wp-content", "themes", stylesheet, subdir)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var out []HTMLFile
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".html") {
			continue
		}
		slug := strings.TrimSuffix(e.Name(), ".html")
		raw, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		out = append(out, HTMLFile{Slug: slug, Content: string(raw)})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Slug < out[j].Slug })
	return out
}

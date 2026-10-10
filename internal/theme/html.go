// Package theme — HTML block template / template-part file loading (P5.5+).
package theme

import (
	"fmt"
	"os"
	"path/filepath"
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

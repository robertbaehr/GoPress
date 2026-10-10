// Style variations from theme /styles/*.json partials (P5.40).
package theme

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// StyleVariation is a theme.json-like style variation partial
// (WP_Theme_JSON_Resolver::get_style_variations).
type StyleVariation struct {
	Slug       string         `json:"slug,omitempty"`
	Title      string         `json:"title,omitempty"`
	BlockTypes []string       `json:"blockTypes,omitempty"`
	Settings   map[string]any `json:"settings,omitempty"`
	Styles     map[string]any `json:"styles,omitempty"`
	Version    int            `json:"version,omitempty"`
	Path       string         `json:"-"`
	Raw        map[string]any `json:"-"`
}

type variationFileShape struct {
	Slug       string         `json:"slug"`
	Title      string         `json:"title"`
	BlockTypes []string       `json:"blockTypes"`
	Settings   map[string]any `json:"settings"`
	Styles     map[string]any `json:"styles"`
	Version    int            `json:"version"`
}

// LoadStyleVariations loads style variation JSON files under
// wp-content/themes/{stylesheet}/styles/ (recursive).
// scope "block" keeps only variations that declare blockTypes (P5.40).
func LoadStyleVariations(absPath, stylesheet, scope string) ([]StyleVariation, error) {
	stylesheet = strings.TrimSpace(stylesheet)
	if stylesheet == "" {
		return nil, fmt.Errorf("theme: empty stylesheet")
	}
	base := filepath.Join(absPath, "wp-content", "themes", stylesheet, "styles")
	entries, err := listJSONFiles(base)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	out := make([]StyleVariation, 0, len(entries))
	for _, path := range entries {
		raw, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var shape variationFileShape
		if err := json.Unmarshal(raw, &shape); err != nil {
			continue
		}
		var rawMap map[string]any
		_ = json.Unmarshal(raw, &rawMap)
		v := StyleVariation{
			Slug:       strings.TrimSpace(shape.Slug),
			Title:      strings.TrimSpace(shape.Title),
			BlockTypes: shape.BlockTypes,
			Settings:   shape.Settings,
			Styles:     shape.Styles,
			Version:    shape.Version,
			Path:       path,
			Raw:        rawMap,
		}
		if v.Title == "" {
			v.Title = strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
		}
		if v.Slug == "" {
			v.Slug = toKebabCase(v.Title)
		}
		if !styleVariationHasScope(v, scope) {
			continue
		}
		out = append(out, v)
	}
	return out, nil
}

func styleVariationHasScope(v StyleVariation, scope string) bool {
	scope = strings.TrimSpace(scope)
	if scope == "" || scope == "theme" {
		// Theme-scoped variations omit blockTypes (color/typography presets, etc.).
		return len(v.BlockTypes) == 0
	}
	if scope == "block" {
		return len(v.BlockTypes) > 0
	}
	return true
}

func listJSONFiles(dir string) ([]string, error) {
	var out []string
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if strings.EqualFold(filepath.Ext(path), ".json") {
			out = append(out, path)
		}
		return nil
	})
	return out, err
}

func toKebabCase(s string) string {
	s = strings.TrimSpace(strings.ToLower(s))
	var b strings.Builder
	prevDash := false
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			prevDash = false
			continue
		}
		if !prevDash {
			b.WriteByte('-')
			prevDash = true
		}
	}
	return strings.Trim(b.String(), "-")
}

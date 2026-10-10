// Package theme loads and parses WordPress theme.json (P5.4).
package theme

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// JSON is a parsed theme.json settings/styles subset.
type JSON struct {
	// Theme is the stylesheet slug (directory under wp-content/themes).
	Theme string `json:"theme"`

	// Version is the theme.json schema version.
	Version int `json:"version,omitempty"`

	// Settings is the theme.json "settings" object (untyped subset).
	Settings map[string]any `json:"settings,omitempty"`

	// Styles is the theme.json "styles" object (untyped subset).
	Styles map[string]any `json:"styles,omitempty"`

	// Path is the absolute path to the loaded theme.json file.
	Path string `json:"-"`

	// FileSHA256 is the sha256 hex of the raw theme.json bytes.
	FileSHA256 string `json:"file_sha256,omitempty"`

	// SettingsStylesSHA256 is sha256 of canonical JSON {"settings":…,"styles":…}
	// for lightweight golden dumps (full WP global CSS merge is out of scope).
	SettingsStylesSHA256 string `json:"settings_styles_sha256,omitempty"`
}

type fileShape struct {
	Version  int            `json:"version"`
	Settings map[string]any `json:"settings"`
	Styles   map[string]any `json:"styles"`
}

// LoadFile reads and parses a theme.json file.
func LoadFile(path string) (*JSON, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("theme: read %s: %w", path, err)
	}
	var shape fileShape
	if err := json.Unmarshal(raw, &shape); err != nil {
		return nil, fmt.Errorf("theme: parse %s: %w", path, err)
	}
	sum := sha256.Sum256(raw)
	out := &JSON{
		Version:    shape.Version,
		Settings:   shape.Settings,
		Styles:     shape.Styles,
		Path:       path,
		FileSHA256: hex.EncodeToString(sum[:]),
	}
	out.SettingsStylesSHA256, err = hashSettingsStyles(out.Settings, out.Styles)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// LoadActive loads theme.json for the active stylesheet under ABSPATH.
// stylesheet is the theme directory name (e.g. "twentytwentytwo").
func LoadActive(absPath, stylesheet string) (*JSON, error) {
	stylesheet = strings.TrimSpace(stylesheet)
	if stylesheet == "" {
		return nil, fmt.Errorf("theme: empty stylesheet")
	}
	path := filepath.Join(absPath, "wp-content", "themes", stylesheet, "theme.json")
	tj, err := LoadFile(path)
	if err != nil {
		return nil, err
	}
	tj.Theme = stylesheet
	return tj, nil
}

// Dump returns a golden-comparable summary (theme + settings/styles + hashes).
func (t *JSON) Dump() map[string]any {
	if t == nil {
		return nil
	}
	return map[string]any{
		"theme":                  t.Theme,
		"version":                t.Version,
		"settings":               t.Settings,
		"styles":                 t.Styles,
		"file_sha256":            t.FileSHA256,
		"settings_styles_sha256": t.SettingsStylesSHA256,
	}
}

func hashSettingsStyles(settings, styles map[string]any) (string, error) {
	payload := map[string]any{
		"settings": settings,
		"styles":   styles,
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("theme: marshal settings/styles: %w", err)
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:]), nil
}

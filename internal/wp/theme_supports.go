// Theme supports API (P9.24) — subset of wp-includes/theme.php
// (add_theme_support / current_theme_supports / get_theme_support).
package wp

import (
	"strings"
	"sync"

	"github.com/robertbaehr/GoPress/internal/theme"
)

// ThemeFeatures holds registered theme support flags (mirrors $_wp_theme_features).
// Bare support is stored as true; support with args is stored as []any{args…}
// matching WP's packing of the variadic $args parameter.
type ThemeFeatures struct {
	mu   sync.RWMutex
	feat map[string]any
}

// NewThemeFeatures returns an empty theme feature registry.
func NewThemeFeatures() *ThemeFeatures {
	return &ThemeFeatures{feat: map[string]any{}}
}

// AddThemeSupport registers a theme feature (add_theme_support).
func AddThemeSupport(c *Context, feature string, args ...any) {
	if c == nil {
		return
	}
	if c.ThemeFeatures == nil {
		c.ThemeFeatures = NewThemeFeatures()
	}
	feature = strings.TrimSpace(feature)
	if feature == "" {
		return
	}
	c.ThemeFeatures.mu.Lock()
	defer c.ThemeFeatures.mu.Unlock()
	if len(args) == 0 {
		c.ThemeFeatures.feat[feature] = true
		return
	}
	// WP special-cases: html5 / post-formats require an array arg.
	c.ThemeFeatures.feat[feature] = args
}

// RemoveThemeSupport unregisters a theme feature.
func RemoveThemeSupport(c *Context, feature string) {
	if c == nil || c.ThemeFeatures == nil {
		return
	}
	feature = strings.TrimSpace(feature)
	c.ThemeFeatures.mu.Lock()
	delete(c.ThemeFeatures.feat, feature)
	c.ThemeFeatures.mu.Unlock()
}

// CurrentThemeSupports reports whether the active theme supports feature
// (current_theme_supports). Optional extra args check html5/post-formats/
// post-thumbnails membership and custom-* sub-keys.
func CurrentThemeSupports(c *Context, feature string, args ...any) bool {
	if c == nil || c.ThemeFeatures == nil {
		return false
	}
	feature = strings.TrimSpace(feature)
	if feature == "custom-header-uploads" {
		return CurrentThemeSupports(c, "custom-header", "uploads")
	}
	c.ThemeFeatures.mu.RLock()
	raw, ok := c.ThemeFeatures.feat[feature]
	c.ThemeFeatures.mu.RUnlock()
	if !ok {
		return false
	}
	if len(args) == 0 {
		return true
	}
	switch feature {
	case "post-thumbnails":
		if raw == true {
			return true
		}
		list := themeSupportStringList(raw)
		needle, _ := args[0].(string)
		for _, s := range list {
			if s == needle {
				return true
			}
		}
		return false
	case "html5", "post-formats":
		list := themeSupportStringList(raw)
		needle, _ := args[0].(string)
		for _, s := range list {
			if s == needle {
				return true
			}
		}
		return false
	case "custom-logo", "custom-header", "custom-background":
		m := themeSupportMap(raw)
		key, _ := args[0].(string)
		v, ok := m[key]
		if !ok {
			return false
		}
		switch t := v.(type) {
		case bool:
			return t
		default:
			return v != nil
		}
	}
	return true
}

// GetThemeSupport returns the raw registered value for feature (get_theme_support).
func GetThemeSupport(c *Context, feature string) any {
	if c == nil || c.ThemeFeatures == nil {
		return false
	}
	c.ThemeFeatures.mu.RLock()
	defer c.ThemeFeatures.mu.RUnlock()
	v, ok := c.ThemeFeatures.feat[feature]
	if !ok {
		return false
	}
	return v
}

func themeSupportStringList(raw any) []string {
	switch t := raw.(type) {
	case []any:
		if len(t) == 0 {
			return nil
		}
		// WP stores add_theme_support('html5', $arr) as [$arr].
		if inner, ok := t[0].([]string); ok {
			return inner
		}
		if inner, ok := t[0].([]any); ok {
			out := make([]string, 0, len(inner))
			for _, v := range inner {
				out = append(out, strings.TrimSpace(toStringSimple(v)))
			}
			return out
		}
		out := make([]string, 0, len(t))
		for _, v := range t {
			out = append(out, strings.TrimSpace(toStringSimple(v)))
		}
		return out
	case []string:
		return t
	default:
		return nil
	}
}

func themeSupportMap(raw any) map[string]any {
	switch t := raw.(type) {
	case []any:
		if len(t) == 0 {
			return nil
		}
		if m, ok := t[0].(map[string]any); ok {
			return m
		}
	case map[string]any:
		return t
	}
	return nil
}

func toStringSimple(v any) string {
	switch t := v.(type) {
	case string:
		return t
	default:
		return ""
	}
}

// RegisterDefaultThemeSupports applies _add_default_theme_supports for block
// themes plus the post-formats / block-templates extras TT5 exposes (P9.24).
func RegisterDefaultThemeSupports(c *Context) {
	if c == nil {
		return
	}
	if c.ThemeFeatures == nil {
		c.ThemeFeatures = NewThemeFeatures()
	}
	abs := ""
	if c.Config != nil {
		abs = c.Config.ABSPATH
	}
	stylesheet := ActiveStylesheet(c)
	if !theme.IsBlockTheme(abs, stylesheet) {
		return
	}
	AddThemeSupport(c, "post-thumbnails")
	AddThemeSupport(c, "responsive-embeds")
	AddThemeSupport(c, "editor-styles")
	AddThemeSupport(c, "html5", []any{
		"comment-form", "comment-list", "search-form", "gallery", "caption", "style", "script",
	})
	AddThemeSupport(c, "automatic-feed-links")
	AddThemeSupport(c, "block-templates")
	AddThemeSupport(c, "widgets-block-editor")
	AddThemeSupport(c, "core-block-patterns")
	AddThemeSupport(c, "editor-style")
	// Block themes register all post formats by default in practice for REST.
	AddThemeSupport(c, "post-formats", []any{
		"aside", "audio", "chat", "gallery", "image", "link", "quote", "status", "video",
	})
}

// ThemeSupportsForREST builds the active theme's theme_supports object for
// GET /wp/v2/themes (P9.24). Keys match get_registered_theme_features() entries
// with show_in_rest; unsupported features use schema defaults (false / [standard]).
func ThemeSupportsForREST(c *Context) map[string]any {
	out := map[string]any{
		"align-wide":                          false,
		"automatic-feed-links":                false,
		"block-templates":                     false,
		"block-template-parts":                false,
		"custom-background":                   false,
		"custom-header":                       false,
		"custom-logo":                         false,
		"customize-selective-refresh-widgets": false,
		"dark-editor-style":                   false,
		"disable-custom-colors":               false,
		"disable-custom-font-sizes":           false,
		"disable-custom-gradients":            false,
		"disable-layout-styles":               false,
		"editor-color-palette":                false,
		"editor-font-sizes":                   false,
		"editor-gradient-presets":             false,
		"editor-spacing-sizes":                false,
		"editor-styles":                       false,
		"html5":                               false,
		"formats":                             []any{"standard"},
		"post-thumbnails":                     false,
		"responsive-embeds":                   false,
		"title-tag":                           false,
		"wp-block-styles":                     false,
	}
	boolFeatures := []string{
		"align-wide", "automatic-feed-links", "block-templates", "block-template-parts",
		"customize-selective-refresh-widgets", "dark-editor-style",
		"disable-custom-colors", "disable-custom-font-sizes", "disable-custom-gradients",
		"disable-layout-styles", "editor-styles", "post-thumbnails", "responsive-embeds",
		"title-tag", "wp-block-styles",
	}
	for _, f := range boolFeatures {
		if CurrentThemeSupports(c, f) {
			out[f] = true
		}
	}
	// Object-typed features stay false unless supported (custom-* omitted detail).
	for _, f := range []string{"custom-background", "custom-header", "custom-logo",
		"editor-color-palette", "editor-font-sizes", "editor-gradient-presets", "editor-spacing-sizes"} {
		if CurrentThemeSupports(c, f) {
			out[f] = true
		}
	}
	if CurrentThemeSupports(c, "html5") {
		list := themeSupportStringList(GetThemeSupport(c, "html5"))
		arr := make([]any, 0, len(list))
		for _, s := range list {
			if s != "" {
				arr = append(arr, s)
			}
		}
		out["html5"] = arr
	}
	if CurrentThemeSupports(c, "post-formats") {
		list := themeSupportStringList(GetThemeSupport(c, "post-formats"))
		arr := []any{"standard"}
		for _, s := range list {
			if s != "" && s != "standard" {
				arr = append(arr, s)
			}
		}
		out["formats"] = arr
	}
	return out
}

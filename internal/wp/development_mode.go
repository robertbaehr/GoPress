// wp_get_development_mode / wp_is_development_mode (P2.70 / P2.71) — subset of wp-includes/load.php.
package wp

import (
	"os"
	"strings"
)

var allowedDevelopmentModes = map[string]bool{
	"core":   true,
	"plugin": true,
	"theme":  true,
	"all":    true,
	"":       true,
}

// GetDevelopmentMode returns the current development mode
// (wp_get_development_mode). Allowed values: core, plugin, theme, all, or
// empty string (disabled). Invalid values fall back to "".
//
// Prefer Config.DevelopmentMode (WP_DEVELOPMENT_MODE define /
// GOPRESS_DEVELOPMENT_MODE / WP_DEVELOPMENT_MODE env via config.Load), then
// the process WP_DEVELOPMENT_MODE variable.
func GetDevelopmentMode(c *Context) string {
	mode := ""
	if c != nil && c.Config != nil {
		mode = strings.TrimSpace(c.Config.DevelopmentMode)
	}
	if mode == "" {
		if v, ok := os.LookupEnv("WP_DEVELOPMENT_MODE"); ok {
			mode = strings.TrimSpace(v)
		}
	}
	mode = strings.ToLower(strings.TrimSpace(mode))
	if !allowedDevelopmentModes[mode] {
		return ""
	}
	return mode
}

// IsDevelopmentMode reports whether the site is in the given development mode
// (wp_is_development_mode, P2.71). mode should be core, plugin, theme, or all.
// When the current mode is "all", every mode check returns true.
//
// Applies the wp_is_development_mode filter when Hooks are available so callers
// can force-enable/disable a mode (e.g. tests / tooling).
func IsDevelopmentMode(c *Context, mode string) bool {
	current := GetDevelopmentMode(c)
	if current == "" {
		return applyIsDevelopmentModeFilter(c, false, mode)
	}
	mode = strings.ToLower(strings.TrimSpace(mode))
	enabled := false
	if current == "all" {
		enabled = true
	} else {
		enabled = mode == current
	}
	return applyIsDevelopmentModeFilter(c, enabled, mode)
}

func applyIsDevelopmentModeFilter(c *Context, enabled bool, mode string) bool {
	if c == nil || c.Hooks == nil {
		return enabled
	}
	v := c.Hooks.ApplyFilters("wp_is_development_mode", enabled, mode)
	switch t := v.(type) {
	case bool:
		return t
	case int:
		return t != 0
	case string:
		switch strings.ToLower(strings.TrimSpace(t)) {
		case "1", "true", "yes", "on":
			return true
		case "0", "false", "no", "off", "":
			return false
		}
	}
	return enabled
}

// ShouldCacheThemeData reports whether theme.json / global styles may use
// persistent caches. Returns false when wp_is_development_mode('theme')
// (mirrors WP theme.json cache bypass, P2.71).
func ShouldCacheThemeData(c *Context) bool {
	return !IsDevelopmentMode(c, "theme")
}

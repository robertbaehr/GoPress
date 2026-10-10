// wp_get_development_mode / wp_is_development_mode (P2.70) — subset of wp-includes/load.php.
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
// (wp_is_development_mode). mode should be core, plugin, theme, or all.
// When the current mode is "all", every mode check returns true.
func IsDevelopmentMode(c *Context, mode string) bool {
	current := GetDevelopmentMode(c)
	if current == "" {
		return false
	}
	mode = strings.ToLower(strings.TrimSpace(mode))
	if current == "all" {
		return true
	}
	return mode == current
}

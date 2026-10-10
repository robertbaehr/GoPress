// wp_debug_mode (P2.83) — subset of wp-includes/load.php.
package wp

import (
	"path/filepath"
	"strings"
)

// DebugModeState is the effective debug configuration after ApplyDebugMode.
type DebugModeState struct {
	// Enabled mirrors WP_DEBUG.
	Enabled bool
	// Display mirrors WP_DEBUG && WP_DEBUG_DISPLAY (constant-level; not request-suppressed).
	Display bool
	// DisplayErrors is whether display_errors would be on after request-type suppression.
	DisplayErrors bool
	// LogPath is the error_log path when logging is enabled; empty when off.
	LogPath string
	// Skipped is true when enable_wp_debug_mode_checks filtered to false.
	Skipped bool
	// Applied is true after ApplyDebugMode has run (including skipped).
	Applied bool
}

// ApplyDebugMode applies WP_DEBUG / WP_DEBUG_DISPLAY / WP_DEBUG_LOG like wp_debug_mode().
// GoPress does not call PHP ini_set; it records effective flags on c.Debug for
// callers (block reentrancy warnings, logging stubs, etc.).
func ApplyDebugMode(c *Context) {
	if c == nil {
		return
	}
	// Filter runs before checks (non-web runtimes may disable).
	if c.Hooks != nil {
		if v := c.Hooks.ApplyFilters("enable_wp_debug_mode_checks", true); v != nil {
			if b, ok := v.(bool); ok && !b {
				c.Debug = DebugModeState{Skipped: true, Applied: true}
				return
			}
		}
	}

	debug := false
	displayConst := true // PHP default-constants.php: WP_DEBUG_DISPLAY defaults true
	logRaw := "false"
	contentDir := "wp-content"
	if c.Config != nil {
		debug = c.Config.Debug
		displayConst = c.Config.DebugDisplay
		if strings.TrimSpace(c.Config.DebugLog) != "" {
			logRaw = c.Config.DebugLog
		}
		if c.Config.ABSPATH != "" {
			contentDir = filepath.Join(strings.TrimRight(c.Config.ABSPATH, "/"), "wp-content")
		}
	}

	st := DebugModeState{
		Enabled: debug,
		Display: debug && displayConst,
		Applied: true,
	}

	displayErrors := false
	if debug {
		if displayConst {
			displayErrors = true
		}
		st.LogPath = resolveDebugLogPath(logRaw, contentDir)
	}

	// Suppress display for REST / ajax / JSON like wp_debug_mode().
	if shouldSuppressDebugDisplay(c) {
		displayErrors = false
	}
	st.DisplayErrors = displayErrors
	c.Debug = st
}

func resolveDebugLogPath(logRaw, contentDir string) string {
	v := strings.TrimSpace(logRaw)
	if v == "" {
		return ""
	}
	low := strings.ToLower(v)
	if low == "false" || low == "0" || low == "off" || low == "no" {
		return ""
	}
	if low == "true" || low == "1" || low == "on" || low == "yes" {
		return filepath.Join(contentDir, "debug.log")
	}
	return v
}

func shouldSuppressDebugDisplay(c *Context) bool {
	if c == nil {
		return false
	}
	if IsServingRESTRequest(c) || IsRESTEndpoint(c) {
		return true
	}
	if DoingAjax(c) || DoingCron(c) {
		return true
	}
	if c.Req != nil && IsJSONRequest(c.Req) {
		return true
	}
	return false
}

// IsDebugDisplay reports WP_DEBUG && WP_DEBUG_DISPLAY (constant-level).
// Used by core/post-content reentrancy warnings (matches PHP $is_debug).
func IsDebugDisplay(c *Context) bool {
	if c == nil {
		return false
	}
	if c.Debug.Applied {
		return c.Debug.Display
	}
	if c.Config == nil || !c.Config.Debug {
		return false
	}
	if c.Config.DebugDisplaySet {
		return c.Config.DebugDisplay
	}
	return c.Config.DebugDisplay // default true
}

// Plugin automatic updates are out of scope for GoPress (P9.16).
// GoPress does not ship wordpress.org update checks or auto-update toggles.
package plugins

import (
	"fmt"
	"strings"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// AutoUpdateGoneNotice explains that plugin auto-updates are unavailable.
const AutoUpdateGoneNotice = `<div class="notice notice-info" id="gopress-plugin-auto-update-gone"><p><strong>Automatic Updates</strong> for plugins are <abbr title="not applicable">N/A</abbr> in GoPress. Plugin auto-update enable/disable and wordpress.org update checks are out of scope.</p></div>`

// RegisterAutoUpdateGoneNotice adds an admin_notices stub (P9.16).
func RegisterAutoUpdateGoneNotice(c *wp.Context) {
	if c == nil || c.Hooks == nil {
		return
	}
	c.Hooks.AddAction("admin_notices", func(args ...any) {
		_ = args
		c.AdminNotice(AutoUpdateGoneNotice)
	}, 8, 0)
}

// CheckAutoUpdateGone registers the auto-update N/A notice.
func CheckAutoUpdateGone(c *wp.Context) {
	RegisterAutoUpdateGoneNotice(c)
}

// RejectAutoUpdate returns an error when a plugin auto-update toggle is requested (P9.16).
func RejectAutoUpdate(action string) error {
	action = strings.TrimSpace(strings.ToLower(action))
	switch action {
	case "enable-auto-update", "disable-auto-update",
		"enable-auto-update-selected", "disable-auto-update-selected",
		"enable-auto-updates", "disable-auto-updates":
		return fmt.Errorf("plugins: Automatic Updates are N/A in GoPress")
	}
	return nil
}

// AutoUpdateColumnGoneNote is shown instead of an Automatic Updates column.
const AutoUpdateColumnGoneNote = `<p class="auto-update-gone-notice"><em>Automatic Updates</em> — <abbr title="not applicable">N/A</abbr> (GoPress).</p>`

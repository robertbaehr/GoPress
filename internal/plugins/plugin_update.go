// Plugin updates from wordpress.org are out of scope for GoPress (P9.17).
// GoPress does not ship update.php upgrade-plugin / update-selected workflows.
package plugins

import (
	"fmt"
	"strings"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// PluginUpdateGoneNotice explains that manual plugin updates are unavailable.
const PluginUpdateGoneNotice = `<div class="notice notice-info" id="gopress-plugin-update-gone"><p><strong>Plugin Updates</strong> from wordpress.org are <abbr title="not applicable">N/A</abbr> in GoPress. Installing or upgrading plugins via <code>update.php</code> is out of scope.</p></div>`

// RegisterPluginUpdateGoneNotice adds an admin_notices stub (P9.17).
func RegisterPluginUpdateGoneNotice(c *wp.Context) {
	if c == nil || c.Hooks == nil {
		return
	}
	c.Hooks.AddAction("admin_notices", func(args ...any) {
		_ = args
		c.AdminNotice(PluginUpdateGoneNotice)
	}, 9, 0)
}

// CheckPluginUpdateGone registers the plugin-update N/A notice.
func CheckPluginUpdateGone(c *wp.Context) {
	RegisterPluginUpdateGoneNotice(c)
}

// RejectPluginUpdate returns an error when a wordpress.org plugin upgrade is requested (P9.17).
func RejectPluginUpdate(action string) error {
	action = strings.TrimSpace(strings.ToLower(action))
	switch action {
	case "upgrade-plugin", "update-plugin", "update-selected",
		"update-selected-plugins", "update_plugins", "do-plugin-upgrade":
		return fmt.Errorf("plugins: Plugin Updates are N/A in GoPress")
	}
	return nil
}

// PluginUpdateRowGoneNote is shown instead of an "Update available" row.
const PluginUpdateRowGoneNote = `<p class="plugin-update-gone-notice"><em>Plugin Updates</em> — <abbr title="not applicable">N/A</abbr> (GoPress; no wordpress.org upgrade checks).</p>`

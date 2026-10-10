// Network (multisite) plugin activation is out of scope for GoPress (P9.11).
// GoPress is single-site only: Network Activate / Network Deactivate are N/A.
package plugins

import (
	"fmt"
	"strings"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// NetworkActivateGoneNotice explains that network activation is unavailable.
const NetworkActivateGoneNotice = `<div class="notice notice-info" id="gopress-plugin-network-gone"><p><strong>Network Activate</strong> is <abbr title="not applicable">N/A</abbr> on this single-site installation. GoPress does not support multisite network activation.</p></div>`

// RegisterNetworkActivateGoneNotice adds an admin_notices stub (P9.11).
func RegisterNetworkActivateGoneNotice(c *wp.Context) {
	if c == nil || c.Hooks == nil {
		return
	}
	c.Hooks.AddAction("admin_notices", func(args ...any) {
		_ = args
		c.AdminNotice(NetworkActivateGoneNotice)
	}, 7, 0)
}

// CheckNetworkActivateGone registers the network-activate N/A notice.
func CheckNetworkActivateGone(c *wp.Context) {
	RegisterNetworkActivateGoneNotice(c)
}

// RejectNetworkActivate returns an error when a network-wide activate/deactivate
// is requested on this single-site install (P9.11).
func RejectNetworkActivate(action, networkWide string) error {
	action = strings.TrimSpace(strings.ToLower(action))
	nw := strings.TrimSpace(strings.ToLower(networkWide))
	if nw == "" || nw == "0" || nw == "false" {
		// Also reject explicit network action names.
		switch action {
		case "network-activate", "networkactivate", "activate-network",
			"network-deactivate", "networkdeactivate", "deactivate-network":
			return fmt.Errorf("plugins: Network Activate is N/A on single-site GoPress")
		}
		return nil
	}
	return fmt.Errorf("plugins: Network Activate is N/A on single-site GoPress")
}

// NetworkOnlyRowNote is shown for plugins with Network: true headers.
const NetworkOnlyRowNote = `<p class="network-only-notice"><em>Network Only</em> — Network Activate is <abbr title="not applicable">N/A</abbr> (single site).</p>`

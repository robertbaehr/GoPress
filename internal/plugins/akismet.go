// Akismet is not bundled with GoPress (P9.8). WordPress.org packages often
// ship wp-content/plugins/akismet/; GoPress intentionally omits it and will
// not load a PHP Akismet drop-in if one is copied into the tree.
package plugins

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// AkismetPluginFiles are WordPress-style plugin basenames for Akismet.
var AkismetPluginFiles = []string{
	"akismet/akismet.php",
	"akismet.php",
}

// IsAkismetPlugin reports whether pluginFile is an Akismet plugin path.
func IsAkismetPlugin(pluginFile string) bool {
	pluginFile = strings.TrimSpace(strings.ReplaceAll(pluginFile, "\\", "/"))
	if pluginFile == "" {
		return false
	}
	lower := strings.ToLower(pluginFile)
	for _, f := range AkismetPluginFiles {
		if lower == f {
			return true
		}
	}
	return strings.HasPrefix(lower, "akismet/")
}

// AkismetPresentOnDisk reports whether an akismet plugin directory/file exists
// under wp-content/plugins (unsupported / not loaded).
func AkismetPresentOnDisk(c *wp.Context) bool {
	dir := ContentDir(c)
	if dir == "" {
		return false
	}
	pluginsDir := filepath.Join(dir, "plugins")
	candidates := []string{
		filepath.Join(pluginsDir, "akismet", "akismet.php"),
		filepath.Join(pluginsDir, "akismet.php"),
	}
	for _, path := range candidates {
		st, err := os.Stat(path)
		if err == nil && !st.IsDir() {
			return true
		}
	}
	return false
}

// AkismetNotBundledNotice is the admin notice HTML when Akismet is absent or present-but-unsupported.
const AkismetNotBundledNotice = `<div class="notice notice-info" id="gopress-akismet-gone"><p>Akismet is <strong>not bundled</strong> with GoPress and is out of scope. Comment spam filtering via Akismet is unavailable.</p></div>`

// RegisterAkismetGoneNotice adds an admin_notices info stub (P9.8).
// Always shown so operators know Akismet is not part of the GoPress distribution.
func RegisterAkismetGoneNotice(c *wp.Context) {
	if c == nil || c.Hooks == nil {
		return
	}
	msg := AkismetNotBundledNotice
	if AkismetPresentOnDisk(c) {
		msg = `<div class="notice notice-warning" id="gopress-akismet-gone"><p>Akismet files were detected under <code>wp-content/plugins</code> but GoPress will <strong>not</strong> load them. Akismet is not bundled and remains out of scope.</p></div>`
	}
	c.Hooks.AddAction("admin_notices", func(args ...any) {
		_ = args
		c.AdminNotice(msg)
	}, 6, 0)
}

// CheckAkismetGone registers the not-bundled notice (called from LoadActive).
func CheckAkismetGone(c *wp.Context) {
	RegisterAkismetGoneNotice(c)
}

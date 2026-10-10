// WordPress drop-in detection (P9.3) — _get_dropins subset; files are unsupported.
package plugins

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// DropIn describes a known WordPress wp-content drop-in file.
type DropIn struct {
	File        string // basename under wp-content/
	Description string
	Constant    string // required constant name, or "" when always considered
}

// KnownDropIns mirrors _get_dropins() for single-site (multisite omitted).
var KnownDropIns = []DropIn{
	{File: "advanced-cache.php", Description: "Advanced caching plugin.", Constant: "WP_CACHE"},
	{File: "db.php", Description: "Custom database class."},
	{File: "db-error.php", Description: "Custom database error message."},
	{File: "install.php", Description: "Custom installation script."},
	{File: "maintenance.php", Description: "Custom maintenance message."},
	{File: "object-cache.php", Description: "External object cache."},
	{File: "php-error.php", Description: "Custom PHP error message."},
	{File: "fatal-error-handler.php", Description: "Custom PHP fatal error handler."},
}

// DetectUnsupportedDropIns lists KnownDropIns present under ABSPATH/wp-content.
// GoPress never loads PHP drop-ins; presence is reported as unsupported.
func DetectUnsupportedDropIns(c *wp.Context) ([]DropIn, error) {
	dir := ContentDir(c)
	if dir == "" {
		return nil, nil
	}
	var found []DropIn
	for _, d := range KnownDropIns {
		path := filepath.Join(dir, d.File)
		st, err := os.Stat(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}
		if st.IsDir() {
			continue
		}
		found = append(found, d)
	}
	sort.Slice(found, func(i, j int) bool { return found[i].File < found[j].File })
	return found, nil
}

// ContentDir returns ABSPATH/wp-content (empty when unknown).
func ContentDir(c *wp.Context) string {
	if c == nil || c.Config == nil {
		return ""
	}
	root := strings.TrimSpace(c.Config.ABSPATH)
	if root == "" {
		return ""
	}
	return filepath.Join(root, "wp-content")
}

// RegisterDropInNotices adds an admin_notices warning when unsupported drop-ins exist.
func RegisterDropInNotices(c *wp.Context, dropins []DropIn) {
	if c == nil || c.Hooks == nil || len(dropins) == 0 {
		return
	}
	names := make([]string, len(dropins))
	for i, d := range dropins {
		names[i] = d.File
	}
	msg := fmt.Sprintf(
		`<div class="notice notice-warning" id="gopress-dropins"><p>GoPress detected unsupported WordPress drop-ins and will not load them: <code>%s</code>.</p></div>`,
		strings.Join(names, "</code>, <code>"),
	)
	c.Hooks.AddAction("admin_notices", func(args ...any) {
		_ = args
		c.AdminNotice(msg)
	}, 5, 0)
}

// CheckDropIns detects drop-ins and registers admin notices (called from LoadActive).
func CheckDropIns(c *wp.Context) error {
	found, err := DetectUnsupportedDropIns(c)
	if err != nil {
		return err
	}
	RegisterDropInNotices(c, found)
	return nil
}

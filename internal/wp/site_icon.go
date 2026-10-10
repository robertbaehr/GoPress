// Site icon helpers (P3.31) — subset of get_site_icon_url / has_site_icon / do_favicon.
package wp

import (
	"strings"
)

// DefaultSiteIconPath is the bundled WordPress logo used when no site icon is set.
const DefaultSiteIconPath = "/wp-includes/images/w-logo-gray-white-bg.png"

// GetSiteIconURL returns the site icon URL (get_site_icon_url subset).
// size is accepted for API parity; GoPress currently returns the full attachment URL.
func GetSiteIconURL(c *Context, size int, fallback string) string {
	_ = size
	url := fallback
	if c != nil && c.Options != nil {
		id := int64(0)
		switch v := c.Options.GetOption("site_icon", 0).(type) {
		case int64:
			id = v
		case int:
			id = int64(v)
		case float64:
			id = int64(v)
		case string:
			id = parseInt64Loose(v)
		}
		if id > 0 {
			if att := GetAttachmentURL(c, id); att != "" {
				url = att
			}
		}
	}
	return url
}

// HasSiteIcon reports whether a custom site icon is configured (has_site_icon).
func HasSiteIcon(c *Context) bool {
	return GetSiteIconURL(c, 512, "") != ""
}

// SiteIconRedirectURL is the Location target for do_favicon (32px / default logo).
func SiteIconRedirectURL(c *Context) string {
	home := strings.TrimRight(optionStringValue(c, "home", optionStringValue(c, "siteurl", "")), "/")
	fallback := DefaultSiteIconPath
	if home != "" {
		fallback = home + DefaultSiteIconPath
	}
	return GetSiteIconURL(c, 32, fallback)
}

func parseInt64Loose(s string) int64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	var n int64
	for _, r := range s {
		if r < '0' || r > '9' {
			break
		}
		n = n*10 + int64(r-'0')
	}
	return n
}

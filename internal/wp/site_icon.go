// Site icon helpers (P3.31 / P3.112) — subset of get_site_icon_url / has_site_icon / wp_site_icon / do_favicon.
package wp

import (
	"fmt"
	"strings"
)

// DefaultSiteIconPath is the bundled WordPress logo used when no site icon is set.
const DefaultSiteIconPath = "/wp-includes/images/w-logo-gray-white-bg.png"

// SiteIconID returns the site_icon attachment ID option (0 when unset).
func SiteIconID(c *Context) int64 {
	if c == nil || c.Options == nil {
		return 0
	}
	switch v := c.Options.GetOption("site_icon", 0).(type) {
	case int64:
		return v
	case int:
		return int64(v)
	case float64:
		return int64(v)
	case string:
		return parseInt64Loose(v)
	default:
		return anyToInt64(v)
	}
}

// GetSiteIconURL returns the site icon URL (get_site_icon_url subset, P3.112).
// size >= 512 requests the full attachment; smaller sizes use wp_get_attachment_image_url
// with a square size hint (falls back to full when no intermediate exists).
// Applies the get_site_icon_url filter.
func GetSiteIconURL(c *Context, size int, fallback string) string {
	url := fallback
	id := SiteIconID(c)
	if id > 0 {
		sizeName := "full"
		if size > 0 && size < 512 {
			// Prefer a registered intermediate when present; else full URL.
			sizeName = fmt.Sprintf("%dx%d", size, size)
			if u := GetAttachmentImageURL(c, id, sizeName, false); u != "" {
				url = u
			} else if u := GetAttachmentImageURL(c, id, "full", false); u != "" {
				url = u
			} else if att := GetAttachmentURL(c, id); att != "" {
				url = att
			}
		} else if u := GetAttachmentImageURL(c, id, "full", false); u != "" {
			url = u
		} else if att := GetAttachmentURL(c, id); att != "" {
			url = att
		}
	}
	if c != nil && c.Hooks != nil {
		filtered := c.Hooks.ApplyFilters("get_site_icon_url", url, size, 0)
		if s, ok := filtered.(string); ok {
			url = s
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

// SiteIconMetaTags returns wp_site_icon <link>/<meta> tags for wp_head (P3.112).
// Empty when no custom site icon is set (matches WP early return outside customizer).
func SiteIconMetaTags(c *Context) string {
	if !HasSiteIcon(c) {
		return ""
	}
	meta := make([]string, 0, 4)
	if icon32 := GetSiteIconURL(c, 32, ""); icon32 != "" {
		meta = append(meta, fmt.Sprintf(`<link rel="icon" href="%s" sizes="32x32" />`, EscURL(icon32)))
	}
	if icon192 := GetSiteIconURL(c, 192, ""); icon192 != "" {
		meta = append(meta, fmt.Sprintf(`<link rel="icon" href="%s" sizes="192x192" />`, EscURL(icon192)))
	}
	if icon180 := GetSiteIconURL(c, 180, ""); icon180 != "" {
		meta = append(meta, fmt.Sprintf(`<link rel="apple-touch-icon" href="%s" />`, EscURL(icon180)))
	}
	if icon270 := GetSiteIconURL(c, 270, ""); icon270 != "" {
		meta = append(meta, fmt.Sprintf(`<meta name="msapplication-TileImage" content="%s" />`, EscURL(icon270)))
	}
	if c != nil && c.Hooks != nil {
		filtered := c.Hooks.ApplyFilters("site_icon_meta_tags", meta)
		switch t := filtered.(type) {
		case []string:
			meta = t
		case []any:
			meta = meta[:0]
			for _, v := range t {
				s := strings.TrimSpace(fmt.Sprint(v))
				if s != "" && s != "<nil>" {
					meta = append(meta, s)
				}
			}
		}
	}
	var b strings.Builder
	for _, tag := range meta {
		tag = strings.TrimSpace(tag)
		if tag == "" {
			continue
		}
		b.WriteString(tag)
		if !strings.HasSuffix(tag, "\n") {
			b.WriteByte('\n')
		}
	}
	return b.String()
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

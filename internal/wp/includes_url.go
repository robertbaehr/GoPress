package wp

import "strings"

// IncludesURL returns the wp-includes directory URL with optional path
// (includes_url, P2.163). Applies the includes_url filter.
func IncludesURL(c *Context, path, scheme string) string {
	url := SiteURL(c, "wp-includes/", scheme)
	path = strings.TrimSpace(path)
	if path != "" {
		url = strings.TrimRight(url, "/") + "/" + strings.TrimLeft(path, "/")
	}
	if c != nil && c.Hooks != nil {
		if v := c.Hooks.ApplyFilters("includes_url", url, path, scheme); v != nil {
			if s, ok := v.(string); ok {
				return s
			}
		}
	}
	return url
}

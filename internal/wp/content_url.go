package wp

import "strings"

// ContentURL returns the content directory URL with optional path
// (content_url, P3.160). Base is siteurl/wp-content; scheme via SetURLScheme.
// Applies the content_url filter.
func ContentURL(c *Context, path string) string {
	url := strings.TrimRight(SiteURL(c, "wp-content", ""), "/")
	url = SetURLScheme(c, url, "")
	path = strings.TrimSpace(path)
	if path != "" {
		url += "/" + strings.TrimLeft(path, "/")
	}
	if c != nil && c.Hooks != nil {
		if v := c.Hooks.ApplyFilters("content_url", url, path); v != nil {
			if s, ok := v.(string); ok {
				return s
			}
		}
	}
	return url
}

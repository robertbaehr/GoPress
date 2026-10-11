package wp

import "strings"

// SelfAdminURL returns the admin URL for the current context (self_admin_url, P2.164).
// GoPress is single-site only (no network/user admin), so this matches AdminURL
// and applies the self_admin_url filter. Empty scheme defaults to "admin".
func SelfAdminURL(c *Context, path, scheme string) string {
	if strings.TrimSpace(scheme) == "" {
		scheme = "admin"
	}
	path = strings.TrimLeft(strings.TrimSpace(path), "/")
	adminPath := "wp-admin/"
	if path != "" {
		adminPath = "wp-admin/" + path
	}
	url := SiteURL(c, adminPath, scheme)
	if c != nil && c.Hooks != nil {
		if v := c.Hooks.ApplyFilters("self_admin_url", url, path, scheme); v != nil {
			if s, ok := v.(string); ok {
				return s
			}
		}
	}
	return url
}

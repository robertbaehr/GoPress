package wp

import "strings"

// IsHomeURLUsingHTTPS reports whether home_url uses https (wp_is_home_url_using_https, P2.154).
func IsHomeURLUsingHTTPS(c *Context) bool {
	raw := HomeURL(c, "", "")
	scheme, ok := ParseURLComponent(raw, PHPURLScheme)
	if !ok {
		return false
	}
	s, _ := scheme.(string)
	return strings.EqualFold(strings.TrimSpace(s), "https")
}

// IsSiteURLUsingHTTPS reports whether the siteurl option (via site_url filter) uses https
// (wp_is_site_url_using_https, P2.154).
// Matches WP: filter siteurl option with scheme/path nil so request scheme is not forced.
func IsSiteURLUsingHTTPS(c *Context) bool {
	base := ""
	if c != nil && c.Options != nil {
		base = optionValueString(c.Options.GetOption("siteurl", ""))
	}
	if c != nil && c.Hooks != nil {
		if v := c.Hooks.ApplyFilters("site_url", base, "", nil, nil); v != nil {
			if s, ok := v.(string); ok {
				base = s
			}
		}
	}
	scheme, ok := ParseURLComponent(base, PHPURLScheme)
	if !ok {
		return false
	}
	s, _ := scheme.(string)
	return strings.EqualFold(strings.TrimSpace(s), "https")
}

// IsUsingHTTPS reports whether both home and site URLs use https (wp_is_using_https, P2.154).
func IsUsingHTTPS(c *Context) bool {
	if !IsHomeURLUsingHTTPS(c) {
		return false
	}
	return IsSiteURLUsingHTTPS(c)
}

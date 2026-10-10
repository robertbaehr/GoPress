package wp

import (
	"net/url"
	"regexp"
	"strings"
)

var (
	schemePrefixRE = regexp.MustCompile(`(?i)^[a-z][a-z0-9+.-]*://`)
	schemeHostRE   = regexp.MustCompile(`(?i)^[a-z][a-z0-9+.-]*://[^/]*`)
)

// IsSSL reports whether the current request is HTTPS (is_ssl subset, P3.37).
func IsSSL(c *Context) bool {
	if c == nil || c.Req == nil {
		return false
	}
	if c.Req.TLS != nil {
		return true
	}
	proto := strings.ToLower(strings.TrimSpace(c.Req.Header.Get("X-Forwarded-Proto")))
	if proto == "https" {
		return true
	}
	return c.Req.URL != nil && strings.EqualFold(c.Req.URL.Scheme, "https")
}

// SetURLScheme applies a scheme to url (set_url_scheme, P3.37).
// scheme may be "", "http", "https", "relative", "admin", "login", "login_post", "rest", "rpc".
func SetURLScheme(c *Context, rawURL, scheme string) string {
	orig := scheme
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return ""
	}
	scheme = strings.ToLower(strings.TrimSpace(scheme))
	switch scheme {
	case "":
		if IsSSL(c) {
			scheme = "https"
		} else {
			scheme = "http"
		}
	case "admin", "login", "login_post", "rpc":
		if IsSSL(c) {
			scheme = "https"
		} else {
			scheme = "http"
		}
	case "http", "https", "relative":
		// keep
	default:
		if IsSSL(c) {
			scheme = "https"
		} else {
			scheme = "http"
		}
	}

	out := rawURL
	if strings.HasPrefix(out, "//") {
		out = "http:" + out
	}
	if scheme == "relative" {
		out = schemeHostRE.ReplaceAllString(out, "")
		out = strings.TrimLeft(out, " \t\n\r")
		if out != "" && out[0] == '/' {
			out = "/" + strings.TrimLeft(out, "/ \t\n\r")
		}
	} else {
		if schemePrefixRE.MatchString(out) {
			out = schemePrefixRE.ReplaceAllString(out, scheme+"://")
		} else if strings.HasPrefix(out, "/") {
			// leave path-only URLs alone when no host scheme was present
		} else {
			out = scheme + "://" + strings.TrimPrefix(out, "/")
		}
	}
	if c != nil && c.Hooks != nil {
		if v := c.Hooks.ApplyFilters("set_url_scheme", out, scheme, orig); v != nil {
			if s, ok := v.(string); ok {
				out = s
			}
		}
	}
	return out
}

// HomeURL returns the home URL with optional path (home_url, P3.37).
func HomeURL(c *Context, path, scheme string) string {
	return GetHomeURL(c, path, scheme)
}

// GetHomeURL builds the home URL from the home option (get_home_url, P3.37).
func GetHomeURL(c *Context, path, scheme string) string {
	origScheme := scheme
	base := ""
	if c != nil && c.Options != nil {
		base = optionValueString(c.Options.GetOption("home", ""))
		if base == "" {
			base = optionValueString(c.Options.GetOption("siteurl", ""))
		}
	}
	schemeLower := strings.ToLower(strings.TrimSpace(scheme))
	if schemeLower != "http" && schemeLower != "https" && schemeLower != "relative" {
		if IsSSL(c) {
			scheme = "https"
		} else if u, err := url.Parse(base); err == nil && u.Scheme != "" {
			scheme = u.Scheme
		} else {
			scheme = "http"
		}
	}
	out := SetURLScheme(c, base, scheme)
	if path != "" {
		out = strings.TrimRight(out, "/") + "/" + strings.TrimLeft(path, "/")
	}
	if c != nil && c.Hooks != nil {
		if v := c.Hooks.ApplyFilters("home_url", out, path, origScheme, nil); v != nil {
			if s, ok := v.(string); ok {
				return s
			}
		}
	}
	return out
}

// SiteURL returns the site URL with optional path (site_url, P3.37).
func SiteURL(c *Context, path, scheme string) string {
	return GetSiteURL(c, path, scheme)
}

// GetSiteURL builds the site URL from the siteurl option (get_site_url, P3.37).
func GetSiteURL(c *Context, path, scheme string) string {
	origScheme := scheme
	base := ""
	if c != nil && c.Options != nil {
		base = optionValueString(c.Options.GetOption("siteurl", ""))
	}
	out := SetURLScheme(c, base, scheme)
	if path != "" {
		out = strings.TrimRight(out, "/") + "/" + strings.TrimLeft(path, "/")
	}
	if c != nil && c.Hooks != nil {
		if v := c.Hooks.ApplyFilters("site_url", out, path, origScheme, nil); v != nil {
			if s, ok := v.(string); ok {
				return s
			}
		}
	}
	return out
}

// AdminURL returns site_url for wp-admin (admin_url, P3.37).
// Bare admin_url() includes a trailing slash (…/wp-admin/).
func AdminURL(c *Context, path string) string {
	path = strings.TrimLeft(strings.TrimSpace(path), "/")
	adminPath := "wp-admin/"
	if path != "" {
		adminPath = "wp-admin/" + path
	}
	return SiteURL(c, adminPath, "admin")
}

package wp

import (
	"fmt"
	"net/http"
	"strings"
)

// GetRestURL mirrors get_rest_url (P3.133) for the pretty-permalink root.
// path "/" → "{home}/wp-json/"; other paths append after the prefix.
func GetRestURL(c *Context, path string) string {
	if path == "" {
		path = "/"
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	home := ""
	if c != nil && c.Options != nil {
		siteURL, _ := c.Options.GetOption("siteurl", "").(string)
		home, _ = c.Options.GetOption("home", siteURL).(string)
		home = strings.TrimRight(strings.TrimSpace(home), "/")
		if home == "" {
			home = strings.TrimRight(strings.TrimSpace(siteURL), "/")
		}
	}
	if home == "" {
		return ""
	}
	url := home + "/wp-json"
	if path == "/" {
		url += "/"
	} else {
		url += path
	}
	if c != nil && c.Hooks != nil {
		filtered := c.Hooks.ApplyFilters("rest_url", url, path, nil, "rest")
		if s, ok := filtered.(string); ok && strings.TrimSpace(s) != "" {
			return s
		}
	}
	return url
}

// RestURL mirrors rest_url (P3.133).
func RestURL(c *Context, path string) string {
	return GetRestURL(c, path)
}

// RestOutputLinkHeader emits Link: <api-root>; rel="https://api.w.org/"
// (rest_output_link_header, P3.133). Uses header append semantics like WP.
func RestOutputLinkHeader(w http.ResponseWriter, c *Context) {
	if w == nil {
		return
	}
	root := GetRestURL(c, "/")
	if strings.TrimSpace(root) == "" {
		return
	}
	root = SanitizeURL(root)
	if root == "" {
		return
	}
	ClearPingbackHeaders(w)
	w.Header().Add("Link", fmt.Sprintf(`<%s>; rel="https://api.w.org/"`, root))
}

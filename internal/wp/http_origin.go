package wp

import "strings"

// GetHTTPOrigin returns the HTTP Origin of the current request (get_http_origin, P2.155).
// Reads Origin header via ForRequest context; applies http_origin filter.
func GetHTTPOrigin(c *Context) string {
	origin := ""
	if c != nil && c.Req != nil {
		origin = strings.TrimSpace(c.Req.Header.Get("Origin"))
	}
	if c != nil && c.Hooks != nil {
		if v := c.Hooks.ApplyFilters("http_origin", origin); v != nil {
			if s, ok := v.(string); ok {
				return s
			}
		}
	}
	return origin
}

// HttpSupports reports whether the HTTP transport can satisfy capabilities
// (wp_http_supports subset, P2.155).
// Numeric capability lists are treated as feature flags set true (WP spoof).
// When rawURL is https/ssl and ssl not set, ssl is implied.
// Go always supports ssl; unknown capability keys → false.
func HttpSupports(capabilities map[string]bool, rawURL string) bool {
	caps := map[string]bool{}
	for k, v := range capabilities {
		caps[k] = v
	}
	if rawURL != "" {
		if _, ok := caps["ssl"]; !ok {
			scheme, ok := ParseURLComponent(rawURL, PHPURLScheme)
			if ok {
				if s, _ := scheme.(string); strings.EqualFold(s, "https") || strings.EqualFold(s, "ssl") {
					caps["ssl"] = true
				}
			}
		}
	}
	for k, v := range caps {
		if !v {
			continue
		}
		switch strings.ToLower(strings.TrimSpace(k)) {
		case "ssl":
			// Go net/http always has TLS.
		default:
			return false
		}
	}
	return true
}

// HttpSupportsList is HttpSupports for a numeric list of capability names (P2.155).
func HttpSupportsList(capabilities []string, rawURL string) bool {
	caps := make(map[string]bool, len(capabilities))
	for _, c := range capabilities {
		c = strings.TrimSpace(c)
		if c != "" {
			caps[c] = true
		}
	}
	return HttpSupports(caps, rawURL)
}

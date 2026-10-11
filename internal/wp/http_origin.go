package wp

import (
	"net/url"
	"strings"
)

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

// GetAllowedHTTPOrigins returns http/https origins for admin + home hosts
// (get_allowed_http_origins, P2.156). Applies allowed_http_origins filter.
func GetAllowedHTTPOrigins(c *Context) []string {
	hosts := []string{}
	addHost := func(raw string) {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			return
		}
		u, err := url.Parse(raw)
		if err != nil || u.Host == "" {
			return
		}
		hosts = append(hosts, u.Host)
	}
	addHost(AdminURL(c, ""))
	addHost(HomeURL(c, "", ""))
	seen := map[string]bool{}
	out := make([]string, 0, len(hosts)*2)
	for _, host := range hosts {
		for _, scheme := range []string{"http", "https"} {
			o := scheme + "://" + host
			if seen[o] {
				continue
			}
			seen[o] = true
			out = append(out, o)
		}
	}
	if c != nil && c.Hooks != nil {
		if v := c.Hooks.ApplyFilters("allowed_http_origins", out); v != nil {
			switch t := v.(type) {
			case []string:
				return t
			case []any:
				converted := make([]string, 0, len(t))
				for _, item := range t {
					if s, ok := item.(string); ok {
						converted = append(converted, s)
					}
				}
				return converted
			}
		}
	}
	return out
}

// IsAllowedHTTPOrigin reports whether origin is authorized (is_allowed_http_origin, P2.156).
// Empty originArg uses GetHTTPOrigin. Applies allowed_http_origin filter.
// Returns the origin string when allowed, else "".
func IsAllowedHTTPOrigin(c *Context, originArg string) string {
	origin := strings.TrimSpace(originArg)
	if originArg == "" {
		origin = GetHTTPOrigin(c)
	}
	arg := origin
	if origin != "" {
		allowed := false
		for _, o := range GetAllowedHTTPOrigins(c) {
			if o == origin {
				allowed = true
				break
			}
		}
		if !allowed {
			origin = ""
		}
	}
	if c != nil && c.Hooks != nil {
		if v := c.Hooks.ApplyFilters("allowed_http_origin", origin, arg); v != nil {
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

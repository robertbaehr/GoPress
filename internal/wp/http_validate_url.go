package wp

import (
	"net"
	"strconv"
	"strings"
)

// HttpValidateURL mirrors wp_http_validate_url (P2.146).
// Returns the validated URL string, or "" when invalid (PHP false).
func HttpValidateURL(c *Context, rawURL string) string {
	if rawURL == "" {
		return ""
	}
	// PHP: !is_string || '' === $url || is_numeric($url)
	if isPHPNumericString(rawURL) {
		return ""
	}
	original := rawURL
	cleaned := KsesBadProtocol(rawURL, []string{"http", "https"})
	if cleaned == "" || !strings.EqualFold(cleaned, original) {
		return ""
	}
	parsed, ok := ParseURL(cleaned)
	if !ok || parsed == nil || strings.TrimSpace(parsed.Host) == "" {
		return ""
	}
	if parsed.User != "" || parsed.hasPass {
		return ""
	}
	host := parsed.Host
	if strings.ContainsAny(host, ":#?[]") {
		return ""
	}

	homeHost := ""
	homePort := 0
	homeHasPort := false
	if c != nil && c.Options != nil {
		home := strings.TrimSpace(optionValueString(c.Options.GetOption("home", "")))
		if home != "" {
			if hp, ok := ParseURL(home); ok && hp != nil {
				homeHost = hp.Host
				homePort = hp.Port
				homeHasPort = hp.hasPort
			}
		}
	}
	sameHost := homeHost != "" && strings.EqualFold(homeHost, host)
	hostTrim := strings.Trim(host, ".")

	if !sameHost {
		ip := ""
		if net.ParseIP(hostTrim) != nil && strings.Count(hostTrim, ".") == 3 && !strings.Contains(hostTrim, ":") {
			// IPv4 literal — PHP preg_match dotted-quad path.
			ip = hostTrim
		} else {
			addrs, err := net.LookupIP(hostTrim)
			if err != nil || len(addrs) == 0 {
				return ""
			}
			// Prefer first IPv4 like gethostbyname.
			for _, a := range addrs {
				if v4 := a.To4(); v4 != nil {
					ip = v4.String()
					break
				}
			}
			if ip == "" {
				// No IPv4 — treat as unresolved for this subset (PHP gethostbyname is IPv4-only).
				return ""
			}
		}
		if ip != "" && isRejectedExternalIP(ip) {
			external := false
			if c != nil && c.Hooks != nil {
				if v := c.Hooks.ApplyFilters("http_request_host_is_external", false, hostTrim, cleaned); v != nil {
					if b, ok := v.(bool); ok {
						external = b
					}
				}
			}
			if !external {
				return ""
			}
		}
	}

	if !parsed.hasPort {
		return cleaned
	}
	port := parsed.Port
	allowed := []int{80, 443, 8080}
	if c != nil && c.Hooks != nil {
		if v := c.Hooks.ApplyFilters("http_allowed_safe_ports", allowed, hostTrim, cleaned); v != nil {
			switch t := v.(type) {
			case []int:
				allowed = t
			case []any:
				allowed = nil
				for _, x := range t {
					switch n := x.(type) {
					case int:
						allowed = append(allowed, n)
					case int64:
						allowed = append(allowed, int(n))
					case float64:
						allowed = append(allowed, int(n))
					}
				}
			}
		}
	}
	for _, p := range allowed {
		if p == port {
			return cleaned
		}
	}
	if sameHost && homeHasPort && homePort == port {
		return cleaned
	}
	return ""
}

func isPHPNumericString(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return false
	}
	// PHP is_numeric: integers, floats, scientific — reject those URLs.
	if _, err := strconv.ParseFloat(s, 64); err == nil {
		return true
	}
	return false
}

// isRejectedExternalIP mirrors the private/special IPv4 ranges in wp_http_validate_url.
func isRejectedExternalIP(ip string) bool {
	parts := strings.Split(ip, ".")
	if len(parts) != 4 {
		return false
	}
	var b [4]int
	for i := 0; i < 4; i++ {
		n, err := strconv.Atoi(parts[i])
		if err != nil || n < 0 || n > 255 {
			return false
		}
		b[i] = n
	}
	if b[0] == 127 || b[0] == 10 || b[0] == 0 {
		return true
	}
	if b[0] == 172 && b[1] >= 16 && b[1] <= 31 {
		return true
	}
	if b[0] == 192 && b[1] == 168 {
		return true
	}
	if b[0] == 192 && b[1] == 0 && b[2] == 0 {
		return true
	}
	if b[0] == 192 && b[1] == 0 && b[2] == 2 {
		return true
	}
	if b[0] == 192 && b[1] == 88 && b[2] == 99 {
		return true
	}
	if b[0] == 198 && b[1] == 51 && b[2] == 100 {
		return true
	}
	if b[0] == 203 && b[1] == 0 && b[2] == 113 {
		return true
	}
	if b[0] == 169 && b[1] == 254 {
		return true
	}
	if b[0] == 100 && b[1] >= 64 && b[1] <= 127 {
		return true
	}
	if b[0] == 198 && b[1] >= 18 && b[1] <= 19 {
		return true
	}
	if b[0] >= 224 && b[0] <= 239 {
		return true
	}
	if b[0] >= 240 {
		return true
	}
	return false
}

package wp

import (
	"net"
	"net/url"
	"strconv"
	"strings"
)

// PHP URL component constants (PHP_URL_*).
const (
	PHPURLScheme   = 0
	PHPURLHost     = 1
	PHPURLPort     = 2
	PHPURLUser     = 3
	PHPURLPass     = 4
	PHPURLPath     = 5
	PHPURLQuery    = 6
	PHPURLFragment = 7
)

// ParsedURL is the associative array returned by wp_parse_url (P2.40).
type ParsedURL struct {
	Scheme   string
	Host     string
	Port     int
	User     string
	Pass     string
	Path     string
	Query    string
	Fragment string
	hasPort  bool
}

// ToMap returns PHP-shaped keys present in the parse result.
func (p *ParsedURL) ToMap() map[string]any {
	if p == nil {
		return nil
	}
	out := map[string]any{}
	if p.Scheme != "" {
		out["scheme"] = p.Scheme
	}
	if p.Host != "" {
		out["host"] = p.Host
	}
	if p.hasPort {
		out["port"] = p.Port
	}
	if p.User != "" {
		out["user"] = p.User
	}
	if p.Pass != "" {
		out["pass"] = p.Pass
	}
	if p.Path != "" {
		out["path"] = p.Path
	}
	if p.Query != "" {
		out["query"] = p.Query
	}
	if p.Fragment != "" {
		out["fragment"] = p.Fragment
	}
	return out
}

// ParseURL is wp_parse_url without a component selector (returns all parts).
func ParseURL(raw string) (*ParsedURL, bool) {
	return parseURLRaw(raw)
}

// ParseURLComponent is wp_parse_url($url, $component) (P2.40).
// component -1 returns the full ParsedURL; otherwise a string/int/nil for that key.
// ok is false only when the URL fails to parse (mirrors PHP false).
func ParseURLComponent(raw string, component int) (any, bool) {
	parts, ok := parseURLRaw(raw)
	if !ok {
		return nil, false
	}
	if component < 0 {
		return parts.ToMap(), true
	}
	return getURLComponent(parts, component), true
}

func parseURLRaw(raw string) (*ParsedURL, bool) {
	toUnset := map[string]bool{}
	u := raw
	if strings.HasPrefix(u, "//") {
		toUnset["scheme"] = true
		u = "placeholder:" + u
	} else if strings.HasPrefix(u, "/") {
		toUnset["scheme"] = true
		toUnset["host"] = true
		u = "placeholder://placeholder" + u
	}
	parsed, err := url.Parse(u)
	if err != nil {
		return nil, false
	}
	out := &ParsedURL{}
	if !toUnset["scheme"] {
		out.Scheme = parsed.Scheme
	}
	if !toUnset["host"] {
		out.Host = parsed.Hostname()
	}
	if parsed.Port() != "" {
		if n, err := strconv.Atoi(parsed.Port()); err == nil {
			out.Port = n
			out.hasPort = true
		}
	} else if hostport := parsed.Host; hostport != "" && !toUnset["host"] {
		// url.Parse may leave Host without Port() for non-numeric — ignore.
		if _, port, err := net.SplitHostPort(hostport); err == nil && port != "" {
			if n, err := strconv.Atoi(port); err == nil {
				out.Port = n
				out.hasPort = true
			}
		}
	}
	if parsed.User != nil {
		out.User = parsed.User.Username()
		if pass, ok := parsed.User.Password(); ok {
			out.Pass = pass
		}
	}
	out.Path = parsed.Path
	out.Query = parsed.RawQuery
	out.Fragment = parsed.Fragment
	return out, true
}

func getURLComponent(parts *ParsedURL, component int) any {
	if parts == nil {
		return nil
	}
	switch component {
	case PHPURLScheme:
		if parts.Scheme == "" {
			return nil
		}
		return parts.Scheme
	case PHPURLHost:
		if parts.Host == "" {
			return nil
		}
		return parts.Host
	case PHPURLPort:
		if !parts.hasPort {
			return nil
		}
		return parts.Port
	case PHPURLUser:
		if parts.User == "" {
			return nil
		}
		return parts.User
	case PHPURLPass:
		if parts.Pass == "" {
			return nil
		}
		return parts.Pass
	case PHPURLPath:
		if parts.Path == "" {
			return nil
		}
		return parts.Path
	case PHPURLQuery:
		if parts.Query == "" {
			return nil
		}
		return parts.Query
	case PHPURLFragment:
		if parts.Fragment == "" {
			return nil
		}
		return parts.Fragment
	default:
		return nil
	}
}

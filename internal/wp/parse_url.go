package wp

import (
	"net"
	"net/url"
	"regexp"
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

// ParsedURL is the associative array returned by wp_parse_url (P2.40 / P2.79 / P2.145).
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
	hasPass  bool // true when userinfo included a password (even empty)
	hasPath  bool // true when path key is present (incl. empty "" for empty URL, P2.145)
}

// schemelessHostPort matches PHP parse_url treating host:digits[/path] as authority
// (e.g. "example.com:80/path" → host+port+path, not scheme "example.com").
var schemelessHostPort = regexp.MustCompile(`^(?:([^@/?#]*)@)?([^:/?#]+):(\d+)([/?#].*)?$`)

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
	if p.hasPass {
		out["pass"] = p.Pass
	}
	// P2.145: empty URL yields path:"" (PHP parse_url("")).
	if p.hasPath || p.Path != "" {
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
	// P2.145: PHP parse_url("") → array('path' => '').
	if raw == "" {
		return &ParsedURL{Path: "", hasPath: true}, true
	}
	toUnset := map[string]bool{}
	u := raw
	if strings.HasPrefix(u, "//") {
		toUnset["scheme"] = true
		u = "placeholder:" + u
	} else if strings.HasPrefix(u, "/") {
		toUnset["scheme"] = true
		toUnset["host"] = true
		u = "placeholder://placeholder" + u
	} else if isSchemelessHostPort(u) {
		// P2.79: PHP parse_url("example.com:80/path") → host/port/path.
		toUnset["scheme"] = true
		u = "placeholder://" + u
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
		out.Host = phpParseURLHost(parsed)
	}
	// P2.79: bare "http://" / "http:///path" → false in PHP (http/https require a host).
	if !toUnset["host"] && out.Host == "" {
		switch strings.ToLower(out.Scheme) {
		case "http", "https":
			return nil, false
		}
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
			out.hasPass = true
		}
	}
	out.Path = parsed.Path
	if out.Path != "" {
		out.hasPath = true
	}
	out.Query = parsed.RawQuery
	out.Fragment = parsed.Fragment
	return out, true
}

// phpParseURLHost returns the host the way PHP parse_url does (P2.145).
// IPv6 literals keep square brackets (Hostname() strips them in Go).
func phpParseURLHost(parsed *url.URL) string {
	if parsed == nil {
		return ""
	}
	host := parsed.Host
	if host == "" {
		return ""
	}
	// Bracketed IPv6: "[::1]:8080" or "[::1]".
	if strings.HasPrefix(host, "[") {
		if end := strings.Index(host, "]"); end >= 0 {
			return host[:end+1]
		}
	}
	h := parsed.Hostname()
	if h == "" {
		return ""
	}
	// Re-bracket IPv6 if Hostname() returned bare form.
	if ip := net.ParseIP(h); ip != nil && ip.To4() == nil {
		return "[" + h + "]"
	}
	return h
}

// TranslatePHPURLConstantToKey mirrors _wp_translate_php_url_constant_to_key (P2.145).
// Returns the named array key or "" when the constant is unknown.
func TranslatePHPURLConstantToKey(component int) string {
	switch component {
	case PHPURLScheme:
		return "scheme"
	case PHPURLHost:
		return "host"
	case PHPURLPort:
		return "port"
	case PHPURLUser:
		return "user"
	case PHPURLPass:
		return "pass"
	case PHPURLPath:
		return "path"
	case PHPURLQuery:
		return "query"
	case PHPURLFragment:
		return "fragment"
	default:
		return ""
	}
}

func isSchemelessHostPort(raw string) bool {
	if raw == "" || strings.Contains(raw, "://") {
		return false
	}
	if strings.HasPrefix(raw, "//") || strings.HasPrefix(raw, "/") ||
		strings.HasPrefix(raw, "?") || strings.HasPrefix(raw, "#") {
		return false
	}
	return schemelessHostPort.MatchString(raw)
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
		if !parts.hasPass {
			return nil
		}
		return parts.Pass
	case PHPURLPath:
		if !parts.hasPath && parts.Path == "" {
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

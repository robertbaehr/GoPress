package wp

import (
	"net/http"
	"regexp"
	"strings"
)

// reLinkRelPingback matches a Link header value that advertises pingback
// discovery (rel=pingback with optional quotes/spacing; P3.131 / P3.132).
var reLinkRelPingback = regexp.MustCompile(`(?i)(?:^|[;\s])rel\s*=\s*(?:pingback|"pingback"|'pingback'|"[^"]*\bpingback\b[^"]*"|'[^']*\bpingback\b[^']*')`)

// ClearPingbackHeaders removes X-Pingback and Link rel=pingback headers (P3.131 / P3.132).
// GoPress keeps XML-RPC/pingbacks out of scope (P3.41 / P3.98), so front, REST,
// discovery, and xmlrpc responses must never advertise pingback discovery.
// P3.132: also splits comma-joined Link values and matches spaced/quoted rel tokens.
func ClearPingbackHeaders(w http.ResponseWriter) {
	if w == nil {
		return
	}
	w.Header().Del("X-Pingback")
	vals := w.Header().Values("Link")
	if len(vals) == 0 {
		return
	}
	kept := make([]string, 0, len(vals))
	for _, v := range vals {
		for _, part := range splitLinkHeader(v) {
			if isPingbackLink(part) {
				continue
			}
			kept = append(kept, part)
		}
	}
	w.Header().Del("Link")
	for _, v := range kept {
		w.Header().Add("Link", v)
	}
}

// IsPingbackLink reports whether a single Link header value is pingback discovery (P3.132).
func IsPingbackLink(v string) bool {
	return isPingbackLink(v)
}

func isPingbackLink(v string) bool {
	v = strings.TrimSpace(v)
	if v == "" {
		return false
	}
	lower := strings.ToLower(v)
	if strings.Contains(lower, `rel="pingback"`) || strings.Contains(lower, "rel=pingback") ||
		strings.Contains(lower, "rel='pingback'") {
		return true
	}
	return reLinkRelPingback.MatchString(v)
}

// splitLinkHeader splits a comma-joined Link header into individual values,
// ignoring commas inside <URI> and quoted strings (P3.132).
func splitLinkHeader(v string) []string {
	v = strings.TrimSpace(v)
	if v == "" {
		return nil
	}
	if !strings.Contains(v, ",") {
		return []string{v}
	}
	var parts []string
	var b strings.Builder
	inURI := false
	inQuote := byte(0)
	for i := 0; i < len(v); i++ {
		c := v[i]
		switch {
		case inQuote != 0:
			b.WriteByte(c)
			if c == inQuote && (i == 0 || v[i-1] != '\\') {
				inQuote = 0
			}
		case c == '"' || c == '\'':
			inQuote = c
			b.WriteByte(c)
		case c == '<':
			inURI = true
			b.WriteByte(c)
		case c == '>' && inURI:
			inURI = false
			b.WriteByte(c)
		case c == ',' && !inURI:
			if part := strings.TrimSpace(b.String()); part != "" {
				parts = append(parts, part)
			}
			b.Reset()
		default:
			b.WriteByte(c)
		}
	}
	if part := strings.TrimSpace(b.String()); part != "" {
		parts = append(parts, part)
	}
	if len(parts) == 0 {
		return []string{v}
	}
	return parts
}

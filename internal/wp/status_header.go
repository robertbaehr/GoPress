package wp

import (
	"fmt"
	"net/http"
	"strings"
)

// statusHeaderDescs maps HTTP status codes to WP get_status_header_desc strings (P2.158).
var statusHeaderDescs = map[int]string{
	100: "Continue",
	101: "Switching Protocols",
	102: "Processing",
	103: "Early Hints",
	200: "OK",
	201: "Created",
	202: "Accepted",
	203: "Non-Authoritative Information",
	204: "No Content",
	205: "Reset Content",
	206: "Partial Content",
	207: "Multi-Status",
	226: "IM Used",
	300: "Multiple Choices",
	301: "Moved Permanently",
	302: "Found",
	303: "See Other",
	304: "Not Modified",
	305: "Use Proxy",
	306: "Reserved",
	307: "Temporary Redirect",
	308: "Permanent Redirect",
	400: "Bad Request",
	401: "Unauthorized",
	402: "Payment Required",
	403: "Forbidden",
	404: "Not Found",
	405: "Method Not Allowed",
	406: "Not Acceptable",
	407: "Proxy Authentication Required",
	408: "Request Timeout",
	409: "Conflict",
	410: "Gone",
	411: "Length Required",
	412: "Precondition Failed",
	413: "Request Entity Too Large",
	414: "Request-URI Too Long",
	415: "Unsupported Media Type",
	416: "Requested Range Not Satisfiable",
	417: "Expectation Failed",
	418: "I'm a teapot",
	421: "Misdirected Request",
	422: "Unprocessable Entity",
	423: "Locked",
	424: "Failed Dependency",
	425: "Too Early",
	426: "Upgrade Required",
	428: "Precondition Required",
	429: "Too Many Requests",
	431: "Request Header Fields Too Large",
	451: "Unavailable For Legal Reasons",
	500: "Internal Server Error",
	501: "Not Implemented",
	502: "Bad Gateway",
	503: "Service Unavailable",
	504: "Gateway Timeout",
	505: "HTTP Version Not Supported",
	506: "Variant Also Negotiates",
	507: "Insufficient Storage",
	510: "Not Extended",
	511: "Network Authentication Required",
}

// GetStatusHeaderDesc returns the WP status phrase for code (get_status_header_desc, P2.158).
// Unknown / non-positive codes → "".
func GetStatusHeaderDesc(code int) string {
	if code < 0 {
		code = -code
	}
	return statusHeaderDescs[code]
}

// StatusHeader writes the HTTP status line via ResponseWriter (status_header, P2.158).
// Empty description uses GetStatusHeaderDesc; still-empty description is a no-op.
// Applies status_header filter to the full status line string (protocol code description).
func StatusHeader(w http.ResponseWriter, c *Context, code int, description string) {
	if w == nil {
		return
	}
	description = strings.TrimSpace(description)
	if description == "" {
		description = GetStatusHeaderDesc(code)
	}
	if description == "" {
		return
	}
	protocol := "HTTP/1.1"
	if c != nil && c.Req != nil && c.Req.Proto != "" {
		protocol = c.Req.Proto
	}
	line := fmt.Sprintf("%s %d %s", protocol, code, description)
	if c != nil && c.Hooks != nil {
		if v := c.Hooks.ApplyFilters("status_header", line, code, description, protocol); v != nil {
			if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
				line = s
			}
		}
	}
	// Go's ResponseWriter uses WriteHeader(code); the phrase comes from net/http.
	// Preserve WP filter side effects by storing the line when possible.
	w.Header().Set("X-GoPress-Status-Line", line)
	w.WriteHeader(code)
}

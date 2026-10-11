package wp

import "strings"

// GetServerProtocol returns the request protocol string (wp_get_server_protocol, P2.160).
// Allowed inputs: HTTP/1.1, HTTP/2, HTTP/2.0, HTTP/3. Anything else (incl. empty) → HTTP/1.0.
func GetServerProtocol(c *Context) string {
	proto := ""
	if c != nil && c.Req != nil {
		proto = strings.TrimSpace(c.Req.Proto)
	}
	switch proto {
	case "HTTP/1.1", "HTTP/2", "HTTP/2.0", "HTTP/3":
		return proto
	default:
		return "HTTP/1.0"
	}
}

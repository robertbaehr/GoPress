package wp

import "strings"

// GetServerProtocol returns the request protocol string (wp_get_server_protocol, P2.160).
// Allowed: HTTP/1.0, HTTP/1.1, HTTP/2, HTTP/2.0, HTTP/3. Unknown/empty → HTTP/1.0.
func GetServerProtocol(c *Context) string {
	proto := "HTTP/1.0"
	if c != nil && c.Req != nil && strings.TrimSpace(c.Req.Proto) != "" {
		proto = strings.TrimSpace(c.Req.Proto)
	}
	switch strings.ToUpper(proto) {
	case "HTTP/1.1", "HTTP/2", "HTTP/2.0", "HTTP/3", "HTTP/3.0", "HTTP/1.0":
		return proto
	default:
		return "HTTP/1.0"
	}
}

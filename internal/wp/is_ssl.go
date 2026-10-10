package wp

import (
	"net"
	"net/http"
	"strings"
)

// IsSSL reports whether the current request is HTTPS (is_ssl, P3.37 / P2.81).
//
// Mirrors wp-includes/load.php is_ssl():
//   - TLS connection (Go: Req.TLS)
//   - HTTPS CGI header "on" / "1"
//   - SERVER_PORT / Host port 443
//
// Plus common reverse-proxy signals used by GoPress (X-Forwarded-Proto, URL.Scheme).
func IsSSL(c *Context) bool {
	if c == nil || c.Req == nil {
		return false
	}
	return requestIsSSL(c.Req)
}

func requestIsSSL(r *http.Request) bool {
	if r == nil {
		return false
	}
	if r.TLS != nil {
		return true
	}
	// CGI / FastCGI style HTTPS server variable (P2.81).
	https := strings.TrimSpace(r.Header.Get("HTTPS"))
	if https != "" {
		switch strings.ToLower(https) {
		case "on", "1":
			return true
		}
	}
	// Port 443 via Host / URL (SERVER_PORT parity, P2.81).
	if hostPortIs443(r) {
		return true
	}
	proto := strings.ToLower(strings.TrimSpace(r.Header.Get("X-Forwarded-Proto")))
	if proto == "https" {
		return true
	}
	return r.URL != nil && strings.EqualFold(r.URL.Scheme, "https")
}

func hostPortIs443(r *http.Request) bool {
	if r == nil {
		return false
	}
	candidates := []string{}
	if r.URL != nil && r.URL.Host != "" {
		candidates = append(candidates, r.URL.Host)
	}
	if r.Host != "" {
		candidates = append(candidates, r.Host)
	}
	for _, h := range candidates {
		_, port, err := net.SplitHostPort(h)
		if err != nil {
			continue
		}
		if port == "443" {
			return true
		}
	}
	return false
}

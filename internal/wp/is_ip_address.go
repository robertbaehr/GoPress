package wp

import (
	"net"
	"regexp"
	"strings"
)

// IPv4 dotted-quad shape used by WP_Http::is_ip_address (not a full validity check).
var ipv4AddressPattern = regexp.MustCompile(`^\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}$`)

// IsIPAddress mirrors WP_Http::is_ip_address (P2.47).
// Returns 4 for IPv4-shaped strings, 6 for IPv6-shaped strings, and 0 otherwise.
//
// IPv4 matches the same loose dotted-quad regex as core. IPv6 uses net.ParseIP
// after trimming brackets/spaces (Go's RE2 cannot express WP's lookahead regex).
func IsIPAddress(maybeIP string) int {
	if ipv4AddressPattern.MatchString(maybeIP) {
		return 4
	}
	trimmed := strings.Trim(maybeIP, " []")
	if !strings.Contains(trimmed, ":") {
		return 0
	}
	ip := net.ParseIP(trimmed)
	if ip == nil || ip.To4() != nil {
		return 0
	}
	return 6
}

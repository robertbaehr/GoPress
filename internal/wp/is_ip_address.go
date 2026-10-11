package wp

import (
	"net"
	"regexp"
	"strings"
)

// IPv4 dotted-quad shape used by WP_Http::is_ip_address (not a full validity check).
var ipv4AddressPattern = regexp.MustCompile(`^\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}$`)

// Strict IPv4 used by rest_is_ip_address (each octet 0–255).
var restIPv4AddressPattern = regexp.MustCompile(
	`^(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)$`,
)

// IsIPAddress mirrors WP_Http::is_ip_address (P2.47 / P2.118).
// Returns 4 for IPv4-shaped strings, 6 for IPv6-shaped strings, and 0 otherwise
// (PHP returns false; Go uses 0 as the falsey stand-in).
//
// IPv4 matches the same loose dotted-quad regex as core. IPv6 uses net.ParseIP
// after trimming brackets/spaces (Go's RE2 cannot express WP's lookahead regex).
// P2.118: reject empty/whitespace-only; IPv6 zone IDs (%eth0) stripped before parse.
func IsIPAddress(maybeIP string) int {
	if maybeIP == "" {
		return 0
	}
	if ipv4AddressPattern.MatchString(maybeIP) {
		return 4
	}
	trimmed := strings.Trim(maybeIP, " []")
	if trimmed == "" || !strings.Contains(trimmed, ":") {
		return 0
	}
	// P2.118: drop IPv6 zone identifier (not in WP's regex; ParseIP rejects it).
	if i := strings.IndexByte(trimmed, '%'); i >= 0 {
		trimmed = trimmed[:i]
	}
	ip := net.ParseIP(trimmed)
	if ip == nil || ip.To4() != nil {
		return 0
	}
	return 6
}

// RestIsIPAddress mirrors rest_is_ip_address (P2.118).
// Returns the IP string when valid, or "" when invalid (PHP false).
// IPv4 requires each octet in 0–255; IPv6 uses net.ParseIP (Requests Ipv6::check_ipv6 stand-in).
func RestIsIPAddress(ip string) string {
	ip = strings.TrimSpace(ip)
	if ip == "" {
		return ""
	}
	if restIPv4AddressPattern.MatchString(ip) {
		return ip
	}
	trimmed := strings.Trim(ip, " []")
	if i := strings.IndexByte(trimmed, '%'); i >= 0 {
		trimmed = trimmed[:i]
	}
	parsed := net.ParseIP(trimmed)
	if parsed == nil || parsed.To4() != nil {
		return ""
	}
	return ip
}

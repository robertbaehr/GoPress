// wp_is_hex_color / sanitize_hex_color (P2.52) — subset of wp-includes/formatting.php.
package wp

import (
	"regexp"
	"strings"
)

var reHexColor = regexp.MustCompile(`(?i)^#([0-9a-f]{3}){1,2}$`)

// IsHexColor reports whether color is a 3- or 6-digit hex color with a leading #.
// Mirrors the validation used by sanitize_hex_color / rest_parse_hex_color.
func IsHexColor(color string) bool {
	color = strings.TrimSpace(color)
	if color == "" {
		return false
	}
	return reHexColor.MatchString(color)
}

// SanitizeHexColor returns color when it is a valid #RGB/#RRGGBB value,
// "" for an empty input, or "" when invalid (WP returns null for invalid —
// Go uses "" so callers can treat non-empty as success).
func SanitizeHexColor(color string) string {
	if color == "" {
		return ""
	}
	if IsHexColor(color) {
		return color
	}
	return ""
}

// SanitizeHexColorNoHash returns a valid hex color without the leading #,
// "" for empty input, or "" when invalid.
func SanitizeHexColorNoHash(color string) string {
	color = strings.TrimLeft(color, "#")
	if color == "" {
		return ""
	}
	if IsHexColor("#" + color) {
		return color
	}
	return ""
}

// MaybeHashHexColor ensures a valid hex color is hashed; otherwise returns color untouched.
func MaybeHashHexColor(color string) string {
	unhashed := SanitizeHexColorNoHash(color)
	if unhashed != "" {
		return "#" + unhashed
	}
	return color
}

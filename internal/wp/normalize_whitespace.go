// normalize_whitespace (P2.111) — subset of wp-includes/formatting.php.
package wp

import (
	"regexp"
	"strings"
)

var (
	reNormalizeNewlines = regexp.MustCompile(`\n+`)
	reNormalizeSpaces   = regexp.MustCompile(`[ \t]+`)
)

// NormalizeWhitespace mirrors WordPress normalize_whitespace().
// Trims the string, maps CR to LF, collapses runs of newlines to a single
// newline, and collapses runs of spaces/tabs to a single space.
func NormalizeWhitespace(str string) string {
	str = strings.TrimSpace(str)
	str = strings.ReplaceAll(str, "\r", "\n")
	str = reNormalizeNewlines.ReplaceAllString(str, "\n")
	str = reNormalizeSpaces.ReplaceAllString(str, " ")
	return str
}

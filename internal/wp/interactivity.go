package wp

import (
	"strings"

	"github.com/robertbaehr/GoPress/internal/htmlapi"
)

// ProcessInteractivityHTML is a WordPress Interactivity API stub (P5.21).
// Server-side directive evaluation is out of scope; data-wp-* attributes
// (including data-wp-interactive) pass through unchanged.
func ProcessInteractivityHTML(html string) string {
	if html == "" {
		return html
	}
	// Touch the HTML API so callers can rely on directive attrs being readable.
	p := htmlapi.NewTagProcessor(html)
	for p.NextTag(nil) {
		if _, ok := p.GetAttribute("data-wp-interactive"); ok {
			_ = ok
		}
	}
	return html
}

// HasInteractivityDirectives reports whether html contains any data-wp-* attribute.
func HasInteractivityDirectives(html string) bool {
	return strings.Contains(strings.ToLower(html), "data-wp-")
}

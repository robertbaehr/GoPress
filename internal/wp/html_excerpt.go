package wp

import (
	"regexp"
	"strings"
)

// reHTMLExcerptEntity trims a truncated HTML entity at the end (wp_html_excerpt).
var reHTMLExcerptEntity = regexp.MustCompile(`&[^;\s]{0,6}$`)

// WPHtmlExcerpt mirrors wp_html_excerpt (P2.139).
// Strips tags (remove_breaks), takes the first count characters (runes),
// drops a trailing incomplete entity, and appends more when truncated.
func WPHtmlExcerpt(str string, count int, more string) string {
	if count < 0 {
		count = 0
	}
	str = WPStripAllTags(str, true)
	runes := []rune(str)
	if count > len(runes) {
		count = len(runes)
	}
	excerpt := string(runes[:count])
	excerpt = reHTMLExcerptEntity.ReplaceAllString(excerpt, "")
	if str != excerpt {
		excerpt = strings.TrimSpace(excerpt) + more
	}
	return excerpt
}

// WPHtmlExcerptEllipsis is wp_html_excerpt with the common &hellip; more suffix.
func WPHtmlExcerptEllipsis(str string, count int) string {
	return WPHtmlExcerpt(str, count, "&hellip;")
}

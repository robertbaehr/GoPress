// wp_trim_words (P2.112) — subset of wp-includes/formatting.php.
package wp

import (
	"regexp"
	"strings"
)

var reTrimWordsSplit = regexp.MustCompile(`[\n\r\t ]+`)

// TrimWords mirrors WordPress wp_trim_words() in words mode.
// Strips tags, splits on whitespace runs, and appends more when truncated.
// Pass more="" to omit a suffix; use TrimWordsEllipsis for the PHP null default.
func TrimWords(text string, numWords int, more string) string {
	original := text
	text = WPStripAllTags(text, false)
	if numWords < 0 {
		numWords = 0
	}
	parts := reTrimWordsSplit.Split(text, -1)
	words := make([]string, 0, len(parts))
	for _, p := range parts {
		if p != "" {
			words = append(words, p)
		}
	}
	if len(words) > numWords {
		words = words[:numWords]
		text = strings.Join(words, " ") + more
	} else {
		text = strings.Join(words, " ")
	}
	_ = original
	return text
}

// TrimWordsEllipsis is wp_trim_words( $text, $num_words ) with default &hellip;.
func TrimWordsEllipsis(text string, numWords int) string {
	return TrimWords(text, numWords, "&hellip;")
}

// TrimWordsFiltered applies the wp_trim_words filter after trimming (P2.112).
// Callers wanting the PHP null $more default should pass "&hellip;" (or use TrimWordsEllipsis).
func TrimWordsFiltered(c *Context, text string, numWords int, more string) string {
	original := text
	out := TrimWords(text, numWords, more)
	if c == nil || c.Hooks == nil {
		return out
	}
	filtered := c.Hooks.ApplyFilters("wp_trim_words", out, numWords, more, original)
	if s, ok := filtered.(string); ok {
		return s
	}
	return out
}

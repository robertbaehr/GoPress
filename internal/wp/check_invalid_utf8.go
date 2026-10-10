package wp

import (
	"strings"
	"unicode/utf8"
)

// CheckInvalidUTF8 mirrors wp_check_invalid_utf8 (P2.42).
// When blog_charset is UTF-8 (GoPress default), valid strings pass through;
// invalid inputs return "" unless strip is true, in which case invalid byte
// sequences are replaced with U+FFFD (�), matching wp_scrub_utf8.
func CheckInvalidUTF8(text string, strip bool) string {
	if text == "" {
		return ""
	}
	if utf8.ValidString(text) {
		return text
	}
	if !strip {
		return ""
	}
	return scrubUTF8(text)
}

// scrubUTF8 replaces invalid UTF-8 byte sequences with U+FFFD (wp_scrub_utf8 subset).
func scrubUTF8(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for len(s) > 0 {
		r, size := utf8.DecodeRuneInString(s)
		if r == utf8.RuneError && size == 1 {
			b.WriteRune(utf8.RuneError)
			s = s[1:]
			continue
		}
		b.WriteString(s[:size])
		s = s[size:]
	}
	return b.String()
}

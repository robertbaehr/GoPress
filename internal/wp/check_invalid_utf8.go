package wp

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

// IsUTF8Charset mirrors is_utf8_charset / _is_utf8_charset (P2.137).
// True for "UTF-8" / "UTF8" (case-insensitive, no trim). Non-strings are false.
func IsUTF8Charset(charset any) bool {
	s, ok := charset.(string)
	if !ok {
		return false
	}
	return strings.EqualFold(s, "UTF-8") || strings.EqualFold(s, "UTF8")
}

// BlogIsUTF8Charset reports whether the site blog_charset is UTF-8 (P2.137).
// Missing/empty option defaults to true (GoPress / WP utf8mb4 installs).
func BlogIsUTF8Charset(c *Context) bool {
	if c == nil || c.Options == nil {
		return true
	}
	raw := c.Options.GetOption("blog_charset", "UTF-8")
	s := strings.TrimSpace(fmt.Sprint(raw))
	if s == "" || s == "<nil>" {
		return true
	}
	return IsUTF8Charset(s)
}

// CheckInvalidUTF8 mirrors wp_check_invalid_utf8 (P2.42 / P2.137).
// When blog_charset is UTF-8 (GoPress default), valid strings pass through;
// invalid inputs return "" unless strip is true, in which case invalid byte
// sequences are replaced with U+FFFD (�), matching wp_scrub_utf8.
// P2.137: non-UTF-8 blog_charset skips validation (passthrough), like WP.
func CheckInvalidUTF8(text string, strip bool) string {
	return checkInvalidUTF8(text, strip, true)
}

// CheckInvalidUTF8ForCharset is wp_check_invalid_utf8 with an explicit charset (P2.137).
// When charset is not UTF-8, text is returned unchanged.
func CheckInvalidUTF8ForCharset(text string, strip bool, charset string) string {
	return checkInvalidUTF8(text, strip, IsUTF8Charset(charset))
}

// CheckInvalidUTF8Ctx uses the site blog_charset option (P2.137).
func CheckInvalidUTF8Ctx(c *Context, text string, strip bool) string {
	return checkInvalidUTF8(text, strip, BlogIsUTF8Charset(c))
}

func checkInvalidUTF8(text string, strip, isUTF8 bool) string {
	if text == "" {
		return ""
	}
	// P2.137: non-UTF-8 charsets skip validation (WP returns $text as-is).
	if !isUTF8 {
		return text
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

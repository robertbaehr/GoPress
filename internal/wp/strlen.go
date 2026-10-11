package wp

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

// StrLen returns the byte length of s (PHP strlen parity, P2.143).
func StrLen(s string) int {
	return len(s)
}

// WPStrLen is an alias of StrLen for the wp_strlen naming convention (P2.143).
func WPStrLen(s string) int {
	return StrLen(s)
}

// MBStrLen mirrors _mb_strlen / mb_strlen compat (P2.143).
// When encoding (or blog_charset when encoding is empty and c is set) is UTF-8,
// returns Unicode code-point count; otherwise returns byte length.
func MBStrLen(s string, encoding string) int {
	if encoding == "" {
		encoding = "UTF-8"
	}
	if IsUTF8Charset(encoding) {
		return utf8CodepointCount(s)
	}
	return len(s)
}

// MBStrLenCtx is MBStrLen using blog_charset when encoding is empty (P2.143).
func MBStrLenCtx(c *Context, s string, encoding string) int {
	if strings.TrimSpace(encoding) == "" {
		if c != nil && c.Options != nil {
			raw := c.Options.GetOption("blog_charset", "UTF-8")
			encoding = strings.TrimSpace(fmt.Sprint(raw))
			if encoding == "" || encoding == "<nil>" {
				encoding = "UTF-8"
			}
		} else {
			encoding = "UTF-8"
		}
	}
	return MBStrLen(s, encoding)
}

// utf8CodepointCount mirrors _wp_utf8_codepoint_count for full-string scans (P2.143).
// Valid runes count as one; each maximal invalid byte span counts as one.
func utf8CodepointCount(s string) int {
	if s == "" {
		return 0
	}
	count := 0
	for len(s) > 0 {
		r, size := utf8.DecodeRuneInString(s)
		if r == utf8.RuneError && size == 1 {
			// Invalid lead — consume maximal subpart of one replacement.
			count++
			s = s[1:]
			continue
		}
		count++
		s = s[size:]
	}
	return count
}

// the_content filter pipeline (P5.11) — blocks, shortcodes, texturize.
package wp

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/robertbaehr/GoPress/internal/blocks"
)

// TheContent runs the display pipeline for post_content (the_content subset):
// do_blocks → do_shortcode → wptexturize, then the "the_content" filter.
func TheContent(c *Context, raw string) string {
	out := ApplyContentFilters(c, raw)
	if c != nil && c.Hooks != nil {
		if v := c.Hooks.ApplyFilters("the_content", out); v != nil {
			if s, ok := v.(string); ok {
				out = s
			}
		}
	}
	return out
}

// ApplyContentFilters runs blocks + shortcodes + texturize without the hook.
func ApplyContentFilters(c *Context, raw string) string {
	if raw == "" {
		return ""
	}
	out := blocks.Render(raw)
	out = DoShortcode(c, out)
	out = Texturize(out)
	return out
}

// Texturize is a wptexturize subset used by REST and front the_content (P2.25).
// Covers ellipses, apostrophes, quotes, en/em dashes, and bare ampersands.
func Texturize(s string) string {
	if s == "" {
		return ""
	}
	var b strings.Builder
	b.Grow(len(s) + 32)
	inTag := false
	nextQuoteOpen := true
	for i := 0; i < len(s); {
		r, size := utf8.DecodeRuneInString(s[i:])
		if r == '<' {
			inTag = true
			b.WriteRune(r)
			i += size
			continue
		}
		if r == '>' && inTag {
			inTag = false
			b.WriteRune(r)
			i += size
			continue
		}
		if inTag {
			b.WriteRune(r)
			i += size
			continue
		}

		// Ellipsis.
		if r == '.' && i+2 < len(s) && s[i:i+3] == "..." {
			b.WriteString("&#8230;")
			i += 3
			continue
		}

		// Dashes: --- → emdash; " -- " → emdash; -- → endash.
		if r == '-' {
			if i+2 < len(s) && s[i:i+3] == "---" {
				b.WriteString("&#8212;")
				i += 3
				continue
			}
			if i+1 < len(s) && s[i+1] == '-' {
				prevSpace := i > 0 && (s[i-1] == ' ' || s[i-1] == '\t' || s[i-1] == '\n')
				nextSpace := i+2 < len(s) && (s[i+2] == ' ' || s[i+2] == '\t' || s[i+2] == '\n')
				if prevSpace && nextSpace {
					b.WriteString("&#8212;")
				} else {
					b.WriteString("&#8211;")
				}
				i += 2
				continue
			}
		}

		// Double quotes → curly open/close.
		if r == '"' {
			if nextQuoteOpen {
				b.WriteString("&#8220;")
			} else {
				b.WriteString("&#8221;")
			}
			nextQuoteOpen = !nextQuoteOpen
			i += size
			continue
		}

		// Apostrophe / single quote.
		if r == '\'' {
			b.WriteString("&#8217;")
			i += size
			continue
		}

		// Bare ampersand (not already an entity).
		if r == '&' {
			rest := s[i+size:]
			if !looksLikeEntity(rest) {
				b.WriteString("&#038;")
				i += size
				continue
			}
		}

		if unicode.IsSpace(r) {
			// keep quote open/close tracking across spaces
		} else if r == '.' || r == ',' || r == ';' || r == ':' || r == '!' || r == '?' {
			nextQuoteOpen = true
		}
		b.WriteRune(r)
		i += size
	}
	return b.String()
}

func looksLikeEntity(rest string) bool {
	if rest == "" {
		return false
	}
	if rest[0] == '#' {
		j := 1
		if j < len(rest) && (rest[j] == 'x' || rest[j] == 'X') {
			j++
			for j < len(rest) && isHexByte(rest[j]) {
				j++
			}
		} else {
			for j < len(rest) && rest[j] >= '0' && rest[j] <= '9' {
				j++
			}
		}
		return j > 1 && j < len(rest) && rest[j] == ';'
	}
	j := 0
	for j < len(rest) && ((rest[j] >= 'a' && rest[j] <= 'z') || (rest[j] >= 'A' && rest[j] <= 'Z')) {
		j++
	}
	return j > 0 && j < len(rest) && rest[j] == ';'
}

func isHexByte(c byte) bool {
	return (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')
}

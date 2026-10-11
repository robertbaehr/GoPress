package wp

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// WPStripAllTags mirrors wp_strip_all_tags (P2.31 / P2.107).
//
// Removes <script>/<style> blocks (including their contents), then all HTML
// tags (PHP strip_tags subset: quoted attributes may contain '>'). When
// removeBreaks is true, collapses runs of whitespace to a single space.
func WPStripAllTags(text string, removeBreaks bool) string {
	text = strings.ReplaceAll(text, "\x00", "")
	// PHP: preg_replace( '@<(script|style)[^>]*?>.*?</\\1>@si', '', $text )
	// RE2 lacks backrefs, so apply script and style removals separately.
	text = reScriptBlock.ReplaceAllString(text, "")
	text = reStyleBlock.ReplaceAllString(text, "")
	// P2.107: quote-aware tag strip (PHP strip_tags), not naive <[^>]*>.
	text = stripHTMLTags(text)
	if removeBreaks {
		text = reWhitespace.ReplaceAllString(text, " ")
	}
	return strings.TrimSpace(text)
}

// stripHTMLTags removes HTML/PHP/ASP tags while respecting quoted attribute
// values that may contain '>' (PHP strip_tags behavior).
func stripHTMLTags(text string) string {
	if text == "" || !strings.Contains(text, "<") {
		return text
	}
	var b strings.Builder
	b.Grow(len(text))
	i := 0
	for i < len(text) {
		lt := strings.IndexByte(text[i:], '<')
		if lt < 0 {
			b.WriteString(text[i:])
			break
		}
		lt += i
		b.WriteString(text[i:lt])
		// Comments: <!-- ... -->
		if strings.HasPrefix(text[lt:], "<!--") {
			if end := strings.Index(text[lt+4:], "-->"); end >= 0 {
				i = lt + 4 + end + 3
				continue
			}
			// Unclosed comment: drop rest (strip_tags-ish).
			break
		}
		end := findTagEnd(text, lt+1)
		if end < 0 {
			// No closing >; keep the '<' literally.
			b.WriteByte('<')
			i = lt + 1
			continue
		}
		i = end + 1
	}
	return b.String()
}

// findTagEnd returns the index of the closing '>' for a tag starting at '<',
// or -1 if none. Skips '>' inside "..." or '...' attribute values.
func findTagEnd(text string, start int) int {
	var quote rune
	for i := start; i < len(text); {
		r, size := utf8.DecodeRuneInString(text[i:])
		if r == utf8.RuneError && size == 1 {
			i++
			continue
		}
		if quote != 0 {
			if r == quote {
				quote = 0
			}
			i += size
			continue
		}
		switch r {
		case '"', '\'':
			quote = r
			i += size
		case '>':
			return i
		default:
			// PHP also treats <? ... ?> and <% ... %> as tags.
			if r == '?' || r == '%' {
				// still scan for >
			}
			if unicode.IsSpace(r) || r == '/' || r == '=' || r == '!' ||
				(r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
				(r >= '0' && r <= '9') || r == ':' || r == '-' || r == '_' ||
				r == '.' || r == '[' || r == ']' || r == '?' || r == '%' ||
				r == '#' || r == ';' || r == '&' || r == ',' || r == '+' ||
				r == '(' || r == ')' || r == '*' || r == '@' || r == '\\' {
				i += size
				continue
			}
			i += size
		}
	}
	return -1
}

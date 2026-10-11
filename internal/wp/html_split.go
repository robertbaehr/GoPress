package wp

import (
	"strings"
	"unicode/utf8"
)

// WPHTMLSplit mirrors wp_html_split (P2.49 / P2.109).
// Splits HTML into text and tag/comment/CDATA pieces (preg_split with
// PREG_SPLIT_DELIM_CAPTURE). Consecutive tags yield an empty string between them.
// P2.109: quote-aware tag ends so '>' inside "..." / '...' attributes does not
// terminate the element (PHP strip_tags / real-world markup; complements the
// naive [^>]*>? branch in get_html_split_regex).
func WPHTMLSplit(input string) []string {
	if input == "" {
		return []string{""}
	}
	var out []string
	i := 0
	for i < len(input) {
		lt := strings.IndexByte(input[i:], '<')
		if lt < 0 {
			out = append(out, input[i:])
			return out
		}
		lt += i
		if lt > i {
			out = append(out, input[i:lt])
		} else if len(out) == 0 {
			// Leading delimiter → empty leading text piece (PHP preg_split).
			out = append(out, "")
		} else {
			// Consecutive tags → empty text between delimiters.
			out = append(out, "")
		}
		end := findHTMLTagEnd(input, lt)
		out = append(out, input[lt:end])
		i = end
	}
	// Trailing empty text when input ends on a tag.
	if len(out) > 0 && strings.HasPrefix(out[len(out)-1], "<") {
		out = append(out, "")
	}
	return out
}

// findHTMLTagEnd returns the index after the HTML element/comment/CDATA starting at lt ('<').
func findHTMLTagEnd(s string, lt int) int {
	rest := s[lt+1:]
	if strings.HasPrefix(rest, "!--") {
		// HTML comment: <!-- ... --> (or rest of string if unclosed).
		body := rest[3:]
		if j := strings.Index(body, "-->"); j >= 0 {
			return lt + 1 + 3 + j + 3
		}
		return len(s)
	}
	if strings.HasPrefix(rest, "![CDATA[") {
		body := rest[len("![CDATA["):]
		if j := strings.Index(body, "]]>"); j >= 0 {
			return lt + 1 + len("![CDATA[") + j + 3
		}
		return len(s)
	}
	// P2.109: normal tag — quote-aware scan for '>' (attrs may contain '>').
	if end := findQuotedTagClose(s, lt+1); end >= 0 {
		return end + 1
	}
	return len(s)
}

// findQuotedTagClose returns the index of '>' closing a tag whose content starts
// at start (immediately after '<'), or -1 if none. Skips '>' inside quotes.
func findQuotedTagClose(text string, start int) int {
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
			i += size
		}
	}
	return -1
}

package wp

import "strings"

// WPHTMLSplit mirrors wp_html_split (P2.49).
// Splits HTML into text and tag/comment/CDATA pieces (preg_split with
// PREG_SPLIT_DELIM_CAPTURE). Consecutive tags yield an empty string between them.
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
	// Normal tag: consume through '>' or EOF (WP [^>]*>?).
	if j := strings.IndexByte(rest, '>'); j >= 0 {
		return lt + 1 + j + 1
	}
	return len(s)
}

package wp

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// WPStripAllTags mirrors wp_strip_all_tags (P2.31 / P2.107 / P2.138).
//
// Removes <script>/<style> blocks (including their contents), then all HTML
// tags (PHP strip_tags subset: quoted attributes may contain '>'). When
// removeBreaks is true, collapses runs of whitespace to a single space.
// P2.138: quote-aware script/style open-tag ends (attrs may contain '>').
func WPStripAllTags(text string, removeBreaks bool) string {
	text = strings.ReplaceAll(text, "\x00", "")
	// PHP: preg_replace( '@<(script|style)[^>]*?>.*?</\\1>@si', '', $text )
	// RE2 lacks backrefs; remove script and style separately with quote-aware opens.
	text = stripScriptStyleBlocks(text, "script")
	text = stripScriptStyleBlocks(text, "style")
	// P2.107: quote-aware tag strip (PHP strip_tags), not naive <[^>]*>.
	text = stripHTMLTags(text)
	if removeBreaks {
		text = reWhitespace.ReplaceAllString(text, " ")
	}
	return strings.TrimSpace(text)
}

// WPStripAllTagsDeep recursively strips tags from string leaves (P2.138).
// Mirrors map_deep( $data, 'wp_strip_all_tags' ) with remove_breaks=false.
func WPStripAllTagsDeep(data any) any {
	return MapDeep(data, func(v any) any {
		switch s := v.(type) {
		case string:
			return WPStripAllTags(s, false)
		default:
			return v
		}
	})
}

// stripScriptStyleBlocks removes <tag…>…</tag> pairs with quote-aware open tags (P2.138).
func stripScriptStyleBlocks(text, tag string) string {
	if text == "" || tag == "" {
		return text
	}
	lower := strings.ToLower(text)
	openPrefix := "<" + tag
	closeTag := "</" + tag + ">"
	var b strings.Builder
	b.Grow(len(text))
	i := 0
	for i < len(text) {
		rel := strings.Index(lower[i:], openPrefix)
		if rel < 0 {
			b.WriteString(text[i:])
			break
		}
		start := i + rel
		// Ensure tag boundary: <script… or <script> / <script/ …
		after := start + len(openPrefix)
		if after < len(text) {
			r, _ := utf8.DecodeRuneInString(text[after:])
			if r != '>' && r != '/' && !unicode.IsSpace(r) {
				// e.g. <scripted — not a script tag
				b.WriteString(text[i : after])
				i = after
				continue
			}
		}
		b.WriteString(text[i:start])
		endOpen := findTagEnd(text, after)
		if endOpen < 0 {
			// Unclosed open tag: leave '<' for stripHTMLTags and continue.
			b.WriteByte('<')
			i = start + 1
			continue
		}
		closeRel := strings.Index(lower[endOpen+1:], closeTag)
		if closeRel < 0 {
			// No matching closer (PHP \\1 backref miss): keep open tag for stripHTMLTags.
			b.WriteString(text[start : endOpen+1])
			i = endOpen + 1
			continue
		}
		i = endOpen + 1 + closeRel + len(closeTag)
	}
	return b.String()
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

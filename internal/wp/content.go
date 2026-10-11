// the_content filter pipeline (P5.11 / P3.123) — blocks, shortcodes, texturize, autop.
package wp

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/robertbaehr/GoPress/internal/blocks"
)

// TheContent runs the display pipeline for post_content (the_content subset):
// do_blocks → do_shortcode → wptexturize → (classic wpautop) → capital_P_dangit,
// then the "the_content" filter (P5.11 / P3.123).
// Shortcodes run before texturize so attribute quotes survive (full WP
// wptexturize skips shortcode interiors; this port keeps the safer order).
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

// TheContentDisplay is the_content echo path: filter pipeline + ]]> escape (P3.123).
func TheContentDisplay(c *Context, raw string) string {
	return strings.ReplaceAll(TheContent(c, raw), "]]>", "]]&gt;")
}

// GetTheContent is get_the_content subset for <!--more--> teaser split (P3.123).
// When moreLinkText is non-empty and a more tag exists, appends a simple more link.
func GetTheContent(raw, moreLinkText string, stripTeaser bool) string {
	if raw == "" {
		return ""
	}
	const moreTag = "<!--more-->"
	idx := strings.Index(raw, moreTag)
	if idx < 0 {
		// Also accept <!--more Text-->.
		if i := strings.Index(raw, "<!--more "); i >= 0 {
			if j := strings.Index(raw[i:], "-->"); j >= 0 {
				idx = i
				end := i + j + 3
				teaser := raw[:idx]
				if stripTeaser {
					return strings.TrimSpace(raw[end:])
				}
				rest := raw[end:]
				if moreLinkText == "" {
					return teaser + rest
				}
				return teaser + moreLinkHTML(moreLinkText) + rest
			}
		}
		return raw
	}
	teaser := raw[:idx]
	rest := raw[idx+len(moreTag):]
	if stripTeaser {
		return strings.TrimSpace(rest)
	}
	if moreLinkText == "" {
		return teaser + rest
	}
	return teaser + moreLinkHTML(moreLinkText) + rest
}

func moreLinkHTML(text string) string {
	return `<p><a class="more-link">` + text + `</a></p>`
}

// ApplyContentFilters runs blocks + shortcodes + texturize + classic wpautop +
// capital_P_dangit without the the_content hook (P3.123).
func ApplyContentFilters(c *Context, raw string) string {
	if raw == "" {
		return ""
	}
	hadBlocks := blocks.HasBlocks(raw)
	out := blocks.Render(raw)
	out = DoShortcode(c, out)
	out = Texturize(out)
	if !hadBlocks {
		out = WpAutop(out, true)
	}
	out = CapitalPDangit(out, "")
	return out
}

// WpAutop is a wpautop subset for classic (non-block) content (P3.123).
// Wraps plain text in <p>, converts double newlines to paragraphs, and optional
// single newlines to <br />. Skips when content already has block-level markup.
func WpAutop(text string, br bool) string {
	if strings.TrimSpace(text) == "" {
		return ""
	}
	trimmed := strings.TrimSpace(text)
	if strings.Contains(text, "<p>") || strings.Contains(text, "<p ") ||
		strings.HasPrefix(trimmed, "<div") || strings.HasPrefix(trimmed, "<ul") ||
		strings.HasPrefix(trimmed, "<ol") || strings.HasPrefix(trimmed, "<h") ||
		strings.HasPrefix(trimmed, "<table") || strings.HasPrefix(trimmed, "<blockquote") {
		if !strings.HasSuffix(text, "\n") {
			text += "\n"
		}
		return text
	}
	// Normalize newlines.
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	text = strings.TrimSpace(text) + "\n"
	parts := strings.Split(text, "\n\n")
	var b strings.Builder
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if br {
			part = strings.ReplaceAll(part, "\n", "<br />\n")
		}
		b.WriteString("<p>")
		b.WriteString(part)
		b.WriteString("</p>\n")
	}
	return b.String()
}

// CapitalPDangit is capital_P_dangit (P3.123).
// For title-like filters, replaces all "Wordpress"; otherwise uses cautious
// replacements around spaces/quotes/tags/parens.
func CapitalPDangit(text, currentFilter string) string {
	if text == "" || !strings.Contains(text, "Wordpress") {
		return text
	}
	if currentFilter == "the_title" || currentFilter == "wp_title" || currentFilter == "document_title" {
		return strings.ReplaceAll(text, "Wordpress", "WordPress")
	}
	repl := []struct{ old, new string }{
		{" Wordpress", " WordPress"},
		{"&#8216;Wordpress", "&#8216;WordPress"},
		{"&#8220;Wordpress", "&#8220;WordPress"},
		{">Wordpress", ">WordPress"},
		{"(Wordpress", "(WordPress"},
	}
	for _, r := range repl {
		text = strings.ReplaceAll(text, r.old, r.new)
	}
	return text
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
				// Surrounding "spaces" use wp_spaces_regexp subset (P2.48).
				prevSpace := i > 0 && isTexturizeSpaceByte(s[i-1])
				nextSpace := i+2 < len(s) && isTexturizeSpaceByte(s[i+2])
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

// isTexturizeSpaceByte is the ASCII subset of wp_spaces_regexp used for dash
// decisions in Texturize (full pattern also matches NBSP / &nbsp;).
func isTexturizeSpaceByte(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r'
}

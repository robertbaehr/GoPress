package rest

import (
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/robertbaehr/GoPress/internal/blocks"
)

var (
	reTags       = regexp.MustCompile(`(?s)<[^>]*>`)
	reMultiSpace = regexp.MustCompile(`[\s\x{00a0}]+`)
)

// renderPostContent applies do_blocks + a minimal the_content texturize subset.
func renderPostContent(raw string) string {
	if raw == "" {
		return ""
	}
	out := blocks.Render(raw)
	return restTexturize(out)
}

// restTexturize is a small wptexturize subset for REST content/excerpts.
func restTexturize(s string) string {
	if s == "" {
		return ""
	}
	var b strings.Builder
	b.Grow(len(s) + 16)
	inTag := false
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
		// "..." → &#8230;
		if r == '.' && i+2 < len(s) && s[i:i+3] == "..." {
			b.WriteString("&#8230;")
			i += 3
			continue
		}
		// Apostrophe → &#8217;
		if r == '\'' {
			b.WriteString("&#8217;")
			i += size
			continue
		}
		b.WriteRune(r)
		i += size
	}
	return b.String()
}

// buildExcerpt mirrors wp_trim_excerpt for empty post_excerpt fields.
func buildExcerpt(rawExcerpt, renderedContent string, protected bool) ProtectedRendered {
	if protected {
		return ProtectedRendered{Rendered: "", Protected: true}
	}
	if strings.TrimSpace(rawExcerpt) != "" {
		ex := restTexturize(strings.TrimSpace(rawExcerpt))
		if !strings.HasPrefix(ex, "<") {
			ex = "<p>" + ex + "</p>\n"
		} else if !strings.HasSuffix(ex, "\n") {
			ex += "\n"
		}
		return ProtectedRendered{Rendered: ex, Protected: false}
	}
	text := stripTagsToText(renderedContent)
	text = strings.TrimSpace(reMultiSpace.ReplaceAllString(text, " "))
	if text == "" {
		return ProtectedRendered{Rendered: "", Protected: false}
	}
	words := strings.Fields(text)
	const excerptWords = 55
	truncated := false
	if len(words) > excerptWords {
		words = words[:excerptWords]
		truncated = true
	}
	body := strings.Join(words, " ")
	body = restTexturize(body)
	if truncated {
		body += " [&hellip;]"
	}
	return ProtectedRendered{Rendered: "<p>" + body + "</p>\n", Protected: false}
}

func stripTagsToText(html string) string {
	if html == "" {
		return ""
	}
	// Turn breaks into spaces before stripping.
	s := strings.ReplaceAll(html, "<br>", " ")
	s = strings.ReplaceAll(s, "<br/>", " ")
	s = strings.ReplaceAll(s, "<br />", " ")
	s = strings.ReplaceAll(s, "</p>", " ")
	s = strings.ReplaceAll(s, "</div>", " ")
	s = reTags.ReplaceAllString(s, "")
	s = strings.ReplaceAll(s, "&nbsp;", " ")
	// Decode a few common entities for excerpt word counting; leave others.
	s = strings.ReplaceAll(s, "&#8217;", "'")
	s = strings.ReplaceAll(s, "&#8230;", "…")
	s = strings.ReplaceAll(s, "&hellip;", "…")
	s = strings.ReplaceAll(s, "&amp;", "&")
	s = strings.ReplaceAll(s, "&lt;", "<")
	s = strings.ReplaceAll(s, "&gt;", ">")
	s = strings.ReplaceAll(s, "&quot;", `"`)
	return strings.Map(func(r rune) rune {
		if unicode.IsControl(r) && r != '\n' && r != '\t' {
			return -1
		}
		return r
	}, s)
}

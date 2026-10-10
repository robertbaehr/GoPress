package rest

import (
	"regexp"
	"strings"
	"unicode"

	"github.com/robertbaehr/GoPress/internal/wp"
)

var (
	reTags       = regexp.MustCompile(`(?s)<[^>]*>`)
	reMultiSpace = regexp.MustCompile(`[\s\x{00a0}]+`)
)

// renderPostContentCtx applies the_content with shortcodes/hooks from c.
func renderPostContentCtx(c *wp.Context, raw string) string {
	return wp.TheContent(c, raw)
}

// restTexturize delegates to the shared Texturize helper.
func restTexturize(s string) string {
	return wp.Texturize(s)
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
	// excerpt_remove_blocks subset: drop media-like markup before stripping tags.
	src := removeExcerptExcludedHTML(renderedContent)
	text := stripTagsToText(src)
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

var (
	reFigureBlock = regexp.MustCompile(`(?is)<figure\b[^>]*>.*?</figure>`)
	reVideoAudio  = regexp.MustCompile(`(?is)<(video|audio)\b[^>]*>.*?</(video|audio)>`)
	reFigcaption  = regexp.MustCompile(`(?is)<figcaption\b[^>]*>.*?</figcaption>`)
)

func removeExcerptExcludedHTML(html string) string {
	s := reFigureBlock.ReplaceAllString(html, "")
	s = reVideoAudio.ReplaceAllString(s, "")
	s = reFigcaption.ReplaceAllString(s, "")
	return s
}

func stripTagsToText(html string) string {
	if html == "" {
		return ""
	}
	// <br> joins without a space (WP excerpt for verse fixtures).
	s := strings.ReplaceAll(html, "<br>", "")
	s = strings.ReplaceAll(s, "<br/>", "")
	s = strings.ReplaceAll(s, "<br />", "")
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

package wp

import (
	"strings"
)

// WPTrimExcerpt mirrors wp_trim_excerpt (P2.140).
// When text is non-empty after trim, it is returned through the wp_trim_excerpt filter.
// When empty, content is taken from post, shortcodes/block comments stripped, then
// trimmed with excerpt_length (default 55) and excerpt_more (default " [&hellip;]").
func WPTrimExcerpt(c *Context, text string, post *Post) string {
	rawExcerpt := text
	if strings.TrimSpace(text) == "" {
		content := ""
		if post != nil {
			content = post.Content
		}
		text = content
		text = StripShortcodes(c, text)
		text = stripBlockComments(text)
		// excerpt_remove_footnotes subset: drop core/footnotes block leftovers lightly.
		text = stripFootnotesMarkup(text)
		// Skip the_content here: PHP temporarily unhooks do_blocks / wp_filter_content_tags
		// before applying it; GoPress keeps a strip-only subset for parity stability.
		text = strings.ReplaceAll(text, "]]>", "]]&gt;")

		excerptLength := 55
		if c != nil && c.Hooks != nil {
			if v := c.Hooks.ApplyFilters("excerpt_length", excerptLength); v != nil {
				switch t := v.(type) {
				case int:
					excerptLength = t
				case int64:
					excerptLength = int(t)
				case float64:
					excerptLength = int(t)
				}
			}
		}
		if excerptLength <= 0 {
			excerptLength = 55
		}

		excerptMore := " [&hellip;]"
		if c != nil && c.Hooks != nil {
			if v := c.Hooks.ApplyFilters("excerpt_more", excerptMore); v != nil {
				if s, ok := v.(string); ok {
					excerptMore = s
				}
			}
		}
		text = TrimWords(text, excerptLength, excerptMore)
	}

	if c != nil && c.Hooks != nil {
		if v := c.Hooks.ApplyFilters("wp_trim_excerpt", text, rawExcerpt); v != nil {
			if s, ok := v.(string); ok {
				text = s
			}
		}
	}
	return text
}

// stripBlockComments removes HTML comments including block delimiters (excerpt_remove_blocks lite).
func stripBlockComments(text string) string {
	for {
		start := strings.Index(text, "<!--")
		if start < 0 {
			break
		}
		end := strings.Index(text[start:], "-->")
		if end < 0 {
			text = text[:start]
			break
		}
		text = text[:start] + text[start+end+3:]
	}
	return text
}

// stripFootnotesMarkup drops obvious footnote list markup for excerpt generation (P2.140).
func stripFootnotesMarkup(text string) string {
	// Minimal: remove ol.wp-block-footnotes blocks if present as raw HTML.
	for {
		start := strings.Index(strings.ToLower(text), `<ol class="wp-block-footnotes`)
		if start < 0 {
			start = strings.Index(strings.ToLower(text), `<ol class='wp-block-footnotes`)
		}
		if start < 0 {
			break
		}
		end := strings.Index(strings.ToLower(text[start:]), "</ol>")
		if end < 0 {
			text = text[:start]
			break
		}
		text = text[:start] + text[start+end+5:]
	}
	return text
}
